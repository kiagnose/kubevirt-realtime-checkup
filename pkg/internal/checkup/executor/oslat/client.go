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

package oslat

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"

	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/checkup/executor/console"
)

type consoleExpecter interface {
	SafeExpectBatchWithResponse(expected []expect.Batcher, timeout time.Duration) ([]expect.BatchRes, error)
}

type Client struct {
	consoleExpecter consoleExpecter
	testDuration    time.Duration
}

func NewClient(vmiUnderTestConsoleExpecter consoleExpecter, testDuration time.Duration) *Client {
	return &Client{
		consoleExpecter: vmiUnderTestConsoleExpecter,
		testDuration:    testDuration,
	}
}

func (t Client) Run(ctx context.Context) (time.Duration, error) {
	type result struct {
		stdout string
		err    error
	}

	resultCh := make(chan result)
	go func() {
		defer close(resultCh)
		const testTimeoutGrace = 5 * time.Minute

		oslatCmd := buildOslatCmd(t.testDuration)

		resp, err := t.consoleExpecter.SafeExpectBatchWithResponse([]expect.Batcher{
			&expect.BSnd{S: oslatCmd + "\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: console.PromptExpression},
		},
			t.testDuration+testTimeoutGrace,
		)
		if err != nil {
			resultCh <- result{"", err}
			return
		}

		exitCode, err := getExitCode(resp[1].Output)
		if err != nil {
			resultCh <- result{"", fmt.Errorf("oslat test failed to get exit code: %w", err)}
			return
		}
		stdout := resp[0].Output
		const successExitCode = 0
		if exitCode != successExitCode {
			log.Printf("oslat test returned exit code: %d. stdout: %s", exitCode, stdout)
			resultCh <- result{stdout, fmt.Errorf("oslat test failed with exit code: %d. See logs for more information", exitCode)}
			return
		}

		resultCh <- result{stdout, nil}
	}()

	var res result
	select {
	case res = <-resultCh:
		if res.err != nil {
			return 0, res.err
		}
	case <-ctx.Done():
		return 0, fmt.Errorf("oslat test canceled due to context closing: %w", ctx.Err())
	}

	log.Printf("Oslat test completed:\n%v", res.stdout)
	return parseMaxLatency(res.stdout)
}

func getExitCode(returnVal string) (int, error) {
	pattern := `\r\n(\d+)\r\n`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(returnVal)

	const minExpectedMatches = 2
	if len(matches) < minExpectedMatches {
		return 0, fmt.Errorf("failed to parse exit value")
	}

	exitCode, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	return exitCode, nil
}

func parseMaxLatency(oslatOutput string) (time.Duration, error) {
	const maximumKeyword = "Maximum"

	maximumEntryLine, err := getResultEntryByKey(oslatOutput, maximumKeyword)
	if err != nil {
		return 0, err
	}

	maxLatencyValues, units, err := parseMaxEntryLine(maximumEntryLine)
	if err != nil {
		return 0, err
	}

	return getMaxLatencyValue(maxLatencyValues, units)
}

func getResultEntryByKey(input, entryKey string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, entryKey) {
			continue
		}
		return line, nil
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return "", scanErr
	}
	return "", fmt.Errorf("failed parsing maximum latency from oslat results")
}

func extractUnits(line string) (lineWithoutUnits, units string, err error) {
	re := regexp.MustCompile(`\((.+?)\)`)
	matches := re.FindStringSubmatch(line)

	const minExpectedMatches = 2
	if len(matches) < minExpectedMatches {
		return "", "", fmt.Errorf("units not found in line: %s", line)
	}

	units = matches[1]
	lineWithoutUnits = strings.Replace(line, matches[0], "", 1)
	return lineWithoutUnits, units, nil
}

func parseMaxEntryLine(maximumEntryLine string) (values []string, units string, err error) {
	const keyValDelimiter = ":"
	var keyWithValues string
	keyWithValues, units, err = extractUnits(maximumEntryLine)
	if err != nil {
		return nil, "", fmt.Errorf("failed to extract units: %w", err)
	}
	keyWithValuesSlice := strings.Split(keyWithValues, keyValDelimiter)
	return strings.Fields(keyWithValuesSlice[1]), units, nil
}

func getMaxLatencyValue(values []string, units string) (time.Duration, error) {
	var coreMaxLatencyDuration time.Duration
	var maxCoresLatencyDuration time.Duration
	var err error
	for _, coreMaxLatencyStr := range values {
		corMaxLatencyWithUnits := coreMaxLatencyStr + units
		if coreMaxLatencyDuration, err = time.ParseDuration(corMaxLatencyWithUnits); err != nil {
			return 0, fmt.Errorf("failed to parse core maximum latency %s: %w", corMaxLatencyWithUnits, err)
		}
		if coreMaxLatencyDuration > maxCoresLatencyDuration {
			maxCoresLatencyDuration = coreMaxLatencyDuration
		}
	}
	return maxCoresLatencyDuration, nil
}

func buildOslatCmd(testDuration time.Duration) string {
	const (
		cpuList          = "2-3"
		realtimePriority = "1"
		workload         = "memmove"
		workloadMemory   = "4K"
	)

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("taskset -c %s ", cpuList))
	sb.WriteString("oslat ")
	sb.WriteString(fmt.Sprintf("--cpu-list %s ", cpuList))
	sb.WriteString(fmt.Sprintf("--rtprio %s ", realtimePriority))
	sb.WriteString(fmt.Sprintf("--duration %s ", testDuration.String()))
	sb.WriteString(fmt.Sprintf("--workload %s ", workload))
	sb.WriteString(fmt.Sprintf("--workload-mem %s ", workloadMemory))

	return sb.String()
}
