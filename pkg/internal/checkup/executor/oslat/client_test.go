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

package oslat_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	expect "github.com/google/goexpect"
	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/checkup/executor/console"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/checkup/executor/oslat"
)

const oslatTestDuration = time.Minute

func TestRunSuccess(t *testing.T) {
	expecter := &expecterStub{
		injectedActualMaxResults: "27 56 (us)",
	}

	oslatClient := oslat.NewClient(
		expecter,
		oslatTestDuration,
	)

	maxLatency, err := oslatClient.Run(context.Background())
	assert.NoError(t, err, "Run returned an error")
	expected := 56 * time.Microsecond
	assert.Equal(t, expected, maxLatency, "Run returned unexpected result")
}

func TestRunFailure(t *testing.T) {
	t.Run("when console returns batch error", func(t *testing.T) {
		expectedBatchErr := errors.New("some error")
		expecter := &expecterStub{
			expectBatchFailureErr: expectedBatchErr,
		}
		oslatClient := oslat.NewClient(
			expecter,
			oslatTestDuration,
		)

		_, err := oslatClient.Run(context.Background())
		assert.ErrorContains(t, err, expectedBatchErr.Error())
	})
	t.Run("when run command returns non-success return value", func(t *testing.T) {
		expectedRunErr := errors.New("oslat test failed with exit code")
		expecter := &expecterStub{
			expectRunFailureErr: expectedRunErr,
		}
		oslatClient := oslat.NewClient(
			expecter,
			oslatTestDuration,
		)

		_, err := oslatClient.Run(context.Background())
		assert.ErrorContains(t, err, expectedRunErr.Error())
	})
	t.Run("when batch times out", func(t *testing.T) {
		expectedTimeoutErr := errors.New("run failed due to timeout")
		expecter := &expecterStub{
			batchRunTimeoutErr: expectedTimeoutErr,
		}
		oslatClient := oslat.NewClient(
			expecter,
			oslatTestDuration,
		)

		_, err := oslatClient.Run(context.Background())
		assert.ErrorContains(t, err, expectedTimeoutErr.Error())
	})
	t.Run("when oslat returns invalid data", func(t *testing.T) {
		expectedInvalidOslatOutputErr := errors.New("failed parsing maximum latency from oslat results")
		oslatClient := oslat.NewClient(
			&expecterStub{
				expectRunInvalidOutput: true,
			},
			oslatTestDuration,
		)

		_, err := oslatClient.Run(context.Background())
		assert.ErrorContains(t, err, expectedInvalidOslatOutputErr.Error())
	})
	t.Run("when checkup context times out", func(t *testing.T) {
		expectedCheckupTimeoutErr := errors.New("oslat test canceled due to context closing")
		oslatClient := oslat.NewClient(
			&expecterStub{},
			oslatTestDuration,
		)

		fakeClock := newFakeClock()
		exceededDeadline := fakeClock.Now().Add(-time.Second)
		ctx, cancel := context.WithDeadline(context.Background(), exceededDeadline)
		defer cancel()

		_, err := oslatClient.Run(ctx)
		assert.ErrorContains(t, err, expectedCheckupTimeoutErr.Error())
	})
}

// fakeClock is a custom fake clock implementation
type fakeClock struct {
	current time.Time
}

func newFakeClock() fakeClock {
	return fakeClock{
		current: time.Now(),
	}
}

// Now returns the current time of the fake clock
func (c fakeClock) Now() time.Time {
	return c.current
}

const (
	oslatRunCmd             = "taskset -c 1-2 oslat --cpu-list 1-2 --rtprio 1 --duration 1m0s --workload memmove --workload-mem 4K \n"
	oslatRunResultsTemplate = "oslat V 2.60\n" +
		"Total runtime: \t\t60 seconds\n" +
		"Thread priority: \tSCHED_FIFO:1\n" +
		"CPU list: \t\t1-2\n" +
		"CPU for main thread: \t0\n" +
		"Workload: \t\tmemmove\n" +
		"Workload mem: \t\t4 (KiB)\n" +
		"Preheat cores: \t\t2\n" +
		"\n" +
		"Pre-heat for 1 seconds...\n" +
		"Test starts...\n" +
		"Test completed.\n" +
		"\n" +
		"        Core:\t 1 2\n" +
		"Counter Freq:\t 2096 2096 (Mhz)\n" +
		"    001 (us):\t 0 0\n" +
		"    002 (us):\t 582681699 615399319\n" +
		"    003 (us):\t 24 28\n" +
		"    004 (us):\t 23 18\n" +
		"    005 (us):\t 13 2805\n" +
		"    006 (us):\t 10492 21962\n" +
		"    007 (us):\t 19863 6356\n" +
		"    008 (us):\t 10473 12481\n" +
		"    009 (us):\t 11218 11417\n" +
		"    010 (us):\t 4928 3433\n" +
		"    011 (us):\t 1443 880\n" +
		"    012 (us):\t 524 211\n" +
		"    013 (us):\t 239 84\n" +
		"    014 (us):\t 178 41\n" +
		"    015 (us):\t 170 39\n" +
		"    016 (us):\t 191 74\n" +
		"    017 (us):\t 109 66\n" +
		"    018 (us):\t 33 56\n" +
		"    019 (us):\t 36 47\n" +
		"    020 (us):\t 52 23\n" +
		"    021 (us):\t 35 17\n" +
		"    022 (us):\t 15 10\n" +
		"    023 (us):\t 6 8\n" +
		"    024 (us):\t 0 5\n" +
		"    025 (us):\t 2 4\n" +
		"    026 (us):\t 0 1\n" +
		"    027 (us):\t 0 1\n" +
		"    028 (us):\t 1 1\n" +
		"    029 (us):\t 0 0\n" +
		"    030 (us):\t 0 1\n" +
		"    031 (us):\t 0 0\n" +
		"    032 (us):\t 0 2 (including overflows)\n" +
		"     Minimum:\t 1 1 (us)\n" +
		"     Average:\t 2.001 2.001 (us)\n" +
		"     Maximum:\t %s\n" +
		"     Max-Min:\t 26 55 (us)\n" +
		"    Duration:\t 59.970 59.970 (sec)\n" +
		"\n" +
		"[root@rt-vmi-rw5tr cloud-user]#"

	oslatRunInvalidOutput = "oslat V 2.60\n" +
		"Total runtime: \t\t300 seconds\n" +
		"Thread priority: \tSCHED_FIFO:1\n" +
		"CPU list: \t\t1\n" +
		"CPU for main thread: \t0\n" +
		"Workload: \t\tmemmove\n" +
		"Workload mem: \t\t4 (KiB)\n" +
		"Preheat cores: \t\t1\n" +
		"\n" +
		"Pre-heat for 1 seconds...\n" +
		"Test starts...\n" +
		"Test completed.\n" +
		"\n"
)

type expecterStub struct {
	injectedActualMaxResults string
	batchRunTimeoutErr       error
	expectBatchFailureErr    error
	expectRunFailureErr      error
	expectRunInvalidOutput   bool
}

func generateBatchResponseWithRetval(runStdout string, runRetVal int) []expect.BatchRes {
	return []expect.BatchRes{
		{
			Idx:    1,
			Output: runStdout,
		},
		{
			Idx:    2,
			Output: fmt.Sprintf("%s%d%s", console.CRLF, runRetVal, console.CRLF),
		},
	}
}

func (es expecterStub) SafeExpectBatchWithResponse(expected []expect.Batcher, _ time.Duration) ([]expect.BatchRes, error) {
	const (
		successExitCode = 0
		failureExitCode = 127
	)

	if es.batchRunTimeoutErr != nil {
		return nil, es.batchRunTimeoutErr
	}
	if es.expectBatchFailureErr != nil {
		return nil, es.expectBatchFailureErr
	}

	var batchRes []expect.BatchRes
	switch expected[0].Arg() {
	case oslatRunCmd:
		if es.expectRunFailureErr != nil {
			batchRes = generateBatchResponseWithRetval(es.expectRunFailureErr.Error(), failureExitCode)
		} else if es.expectRunInvalidOutput {
			batchRes = generateBatchResponseWithRetval(oslatRunInvalidOutput, successExitCode)
		} else {
			oslatOutput := fmt.Sprintf(oslatRunResultsTemplate, es.injectedActualMaxResults)
			batchRes = generateBatchResponseWithRetval(oslatOutput, successExitCode)
		}

	default:
		return nil, fmt.Errorf("command not recognized: %q", expected[0].Arg())
	}

	return batchRes, nil
}
