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

package checkup_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/checkup"
	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/config"
	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/status"
)

const (
	testNamespace      = "target-ns"
	testTargetNodeName = "my-node"
	testPVCName        = "my-rt-vm-pvc"
)

func TestCheckupShouldSucceed(t *testing.T) {
	testClient := newClientStub()
	testCheckup := checkup.New(testClient, testNamespace, newTestConfig())

	assert.NoError(t, testCheckup.Setup(context.Background()))
	assert.NoError(t, testCheckup.Run(context.Background()))
	assert.NoError(t, testCheckup.Teardown(context.Background()))

	actualResults := testCheckup.Results()
	expectedResults := status.Results{}

	assert.Equal(t, expectedResults, actualResults)
}

func TestSetupShouldFail(t *testing.T) {
	t.Run("when VMI creation fails", func(t *testing.T) {
		expectedVMICreationFailure := errors.New("failed to create VMI")

		testClient := newClientStub()
		testClient.vmiCreationFailure = expectedVMICreationFailure
		testCheckup := checkup.New(testClient, testNamespace, newTestConfig())

		assert.ErrorContains(t, testCheckup.Setup(context.Background()), expectedVMICreationFailure.Error())
	})
}

type clientStub struct {
	createdVMIs        map[string]*kvcorev1.VirtualMachineInstance
	vmiCreationFailure error
}

func newClientStub() *clientStub {
	return &clientStub{
		createdVMIs: map[string]*kvcorev1.VirtualMachineInstance{},
	}
}

func (cs *clientStub) CreateVirtualMachineInstance(_ context.Context,
	namespace string,
	vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	if cs.vmiCreationFailure != nil {
		return nil, cs.vmiCreationFailure
	}

	vmiFullName := fmt.Sprintf("%s/%s", namespace, vmi.Name)
	cs.createdVMIs[vmiFullName] = vmi

	return vmi, nil
}

func newTestConfig() config.Config {
	return config.Config{
		PodName:                      "",
		PodUID:                       "",
		TargetNode:                   testTargetNodeName,
		GuestImageSourcePVCNamespace: testNamespace,
		GuestImageSourcePVCName:      testPVCName,
		OslatDuration:                10 * time.Minute,
		OslatLatencyThreshold:        45 * time.Microsecond,
	}
}
