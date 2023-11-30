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

package executor

import (
	"context"
	"time"

	"kubevirt.io/client-go/kubecli"

	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/config"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/status"
)

type vmiSerialConsoleClient interface {
	VMISerialConsole(namespace, name string, timeout time.Duration) (kubecli.StreamInterface, error)
}

type Executor struct {
	vmiSerialClient vmiSerialConsoleClient
	namespace       string
	vmiUsername     string
	vmiPassword     string
}

func New(client vmiSerialConsoleClient, namespace string) Executor {
	return Executor{
		vmiSerialClient: client,
		namespace:       namespace,
		vmiUsername:     config.VMIUsername,
		vmiPassword:     config.VMIPassword,
	}
}

func (e Executor) Execute(ctx context.Context, vmiUnderTestName string) (status.Results, error) {
	return status.Results{}, nil
}