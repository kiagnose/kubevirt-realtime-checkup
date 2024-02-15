/*
 * This file is part of the kiagnose project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package console

import (
	"fmt"
	"log"
	"regexp"
	"time"

	expect "github.com/google/goexpect"
	"google.golang.org/grpc/codes"
)

func (e Expecter) LoginToCentOS(username, password string) error {
	const (
		connectionTimeout = 10 * time.Second
		promptTimeout     = 5 * time.Second
	)

	genExpect, err := e.spawnConsole(connectionTimeout)
	if err != nil {
		return err
	}
	defer genExpect.Close()

	err = genExpect.Send("\n")
	if err != nil {
		return err
	}

	// Do not login, if we already logged in
	loggedInPromptRegex := fmt.Sprintf(
		`(\[%s@(localhost|centos|%s) ~\]\$ |\[root@(localhost|centos|%s) ~\]\# )`, username, e.vmiName, e.vmiName,
	)
	b := []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: loggedInPromptRegex},
	}
	_, err = genExpect.ExpectBatch(b, promptTimeout)
	if err == nil {
		return nil
	}

	b = []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BCas{C: []expect.Caser{
			&expect.Case{
				// Using only "login: " would match things like "Last failed login: Tue Jun  9 22:25:30 UTC 2020 on ttyS0"
				// and in case the VM's did not get hostname form DHCP server try the default hostname
				R:  regexp.MustCompile(fmt.Sprintf(`(localhost|centos|%s) login: `, e.vmiName)),
				S:  fmt.Sprintf("%s\n", username),
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Password:`),
				S:  fmt.Sprintf("%s\n", password),
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Login incorrect`),
				T:  expect.LogContinue("Failed to log in", expect.NewStatus(codes.PermissionDenied, "login failed")),
				Rt: 10,
			},
			&expect.Case{
				R: regexp.MustCompile(loggedInPromptRegex),
				T: expect.OK(),
			},
		}},
	}
	const loginTimeout = 2 * time.Minute
	res, err := genExpect.ExpectBatch(b, loginTimeout)
	if err != nil {
		log.Printf("Login attempt failed: %+v", res)
		// Try once more since sometimes the login prompt is ripped apart by asynchronous daemon updates
		if retryRes, retryErr := genExpect.ExpectBatch(b, 1*time.Minute); retryErr != nil {
			log.Printf("Retried login attempt after two minutes failed: %+v", retryRes)
			return retryErr
		}
	}

	err = configureConsole(genExpect, false)
	if err != nil {
		return err
	}
	return nil
}
