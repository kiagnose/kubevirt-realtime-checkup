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

package config_test

import (
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	kconfig "github.com/kiagnose/kiagnose/kiagnose/config"

	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/config"
)

const (
	testPodName                           = "my-pod"
	testPodUID                            = "0123456789-0123456789"
	testTargetNodeName                    = "my-rt-node"
	testGuestImageSourcePVCNamespace      = "rt-namespace"
	testGuestImageSourcePVCName           = "my-rt-vm"
	testOslatDuration                     = "1h"
	testOslatLatencyThresholdMicroSeconds = "50"
)

func TestNewShouldApplyDefaultsWhenOptionalFieldsAreMissing(t *testing.T) {
	baseConfig := kconfig.Config{
		PodName: testPodName,
		PodUID:  testPodUID,
		Params:  map[string]string{},
	}

	actualConfig, err := config.New(baseConfig)
	assert.NoError(t, err)

	expectedConfig := config.Config{
		PodName:                      testPodName,
		PodUID:                       testPodUID,
		TargetNode:                   "",
		GuestImageSourcePVCNamespace: "",
		GuestImageSourcePVCName:      "",
		OslatDuration:                config.OslatDefaultDuration,
		OslatLatencyThreshold:        config.OslatDefaultLatencyThreshold,
	}
	assert.Equal(t, expectedConfig, actualConfig)
}

func TestNewShouldApplyUserConfig(t *testing.T) {
	baseConfig := kconfig.Config{
		PodName: testPodName,
		PodUID:  testPodUID,
		Params: map[string]string{
			config.TargetNodeParamName:                   testTargetNodeName,
			config.GuestImageSourcePVCNamespaceParamName: testGuestImageSourcePVCNamespace,
			config.GuestImageSourcePVCNameParamName:      testGuestImageSourcePVCName,
			config.OslatDurationParamName:                testOslatDuration,
			config.OslatLatencyThresholdParamName:        testOslatLatencyThresholdMicroSeconds,
		},
	}

	actualConfig, err := config.New(baseConfig)
	assert.NoError(t, err)

	expectedConfig := config.Config{
		PodName:                      testPodName,
		PodUID:                       testPodUID,
		TargetNode:                   testTargetNodeName,
		GuestImageSourcePVCNamespace: testGuestImageSourcePVCNamespace,
		GuestImageSourcePVCName:      testGuestImageSourcePVCName,
		OslatDuration:                time.Hour,
		OslatLatencyThreshold:        50 * time.Microsecond,
	}
	assert.Equal(t, expectedConfig, actualConfig)
}

func TestNewShouldFailWhen(t *testing.T) {
	type failureTestCase struct {
		description    string
		userParameters map[string]string
		expectedError  error
	}

	testCases := []failureTestCase{
		{
			description: "oslatDuration is invalid",
			userParameters: map[string]string{
				config.TargetNodeParamName:                   testTargetNodeName,
				config.GuestImageSourcePVCNamespaceParamName: testGuestImageSourcePVCNamespace,
				config.GuestImageSourcePVCNameParamName:      testGuestImageSourcePVCName,
				config.OslatDurationParamName:                "wrongValue",
				config.OslatLatencyThresholdParamName:        testOslatLatencyThresholdMicroSeconds,
			},
			expectedError: config.ErrInvalidOslatDuration,
		},
		{
			description: "oslatLatencyThresholdMicroSeconds is invalid",
			userParameters: map[string]string{
				config.TargetNodeParamName:                   testTargetNodeName,
				config.GuestImageSourcePVCNamespaceParamName: testGuestImageSourcePVCNamespace,
				config.GuestImageSourcePVCNameParamName:      testGuestImageSourcePVCName,
				config.OslatDurationParamName:                testOslatDuration,
				config.OslatLatencyThresholdParamName:        "wrongValue",
			},
			expectedError: config.ErrInvalidOslatLatencyThreshold,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			baseConfig := kconfig.Config{
				PodName: testPodName,
				PodUID:  testPodUID,
				Params:  testCase.userParameters,
			}

			_, err := config.New(baseConfig)
			assert.ErrorIs(t, err, testCase.expectedError)
		})
	}
}
