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
	"strings"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/checkup"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/config"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/status"
)

const (
	testNamespace      = "target-ns"
	testTargetNodeName = "my-node"
	testPVCName        = "my-rt-vm-pvc"
)

func TestCheckupShouldSucceed(t *testing.T) {
	testClient := newClientStub()
	testCheckup := checkup.New(testClient, testNamespace, newTestConfig(), executorStub{})

	assert.NoError(t, testCheckup.Setup(context.Background()))

	assert.NotEmpty(t, testClient.createdConfigMaps)

	vmiName := testClient.VMIName()
	assert.NotEmpty(t, vmiName)

	assert.NoError(t, testCheckup.Run(context.Background()))
	assert.NoError(t, testCheckup.Teardown(context.Background()))

	_, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, vmiName)
	assert.ErrorContains(t, err, "not found")

	actualResults := testCheckup.Results()
	expectedResults := status.Results{}

	assert.Equal(t, expectedResults, actualResults)
}

func TestSetupShouldFail(t *testing.T) {
	t.Run("when VM under test's ConfigMap creation fails", func(t *testing.T) {
		expectedConfigMapCreationError := errors.New("failed to create ConfigMap")

		testClient := newClientStub()
		testConfig := newTestConfig()
		testClient.configMapCreationFailure = expectedConfigMapCreationError
		testCheckup := checkup.New(testClient, testNamespace, testConfig, executorStub{})

		assert.ErrorContains(t, testCheckup.Setup(context.Background()), expectedConfigMapCreationError.Error())
		assert.Empty(t, testClient.createdVMIs)
	})

	t.Run("when VMI creation fails", func(t *testing.T) {
		expectedVMICreationFailure := errors.New("failed to create VMI")

		testClient := newClientStub()
		testClient.vmiCreationFailure = expectedVMICreationFailure
		testCheckup := checkup.New(testClient, testNamespace, newTestConfig(), executorStub{})

		assert.ErrorContains(t, testCheckup.Setup(context.Background()), expectedVMICreationFailure.Error())
	})

	t.Run("when wait for VMI to boot fails", func(t *testing.T) {
		expectedVMIReadFailure := errors.New("failed to read VMI")

		testClient := newClientStub()
		testConfig := newTestConfig()
		testClient.vmiReadFailure = expectedVMIReadFailure
		testCheckup := checkup.New(testClient, testNamespace, testConfig, executorStub{})

		assert.ErrorContains(t, testCheckup.Setup(context.Background()), expectedVMIReadFailure.Error())
	})
}

func TestTeardownShouldFailWhen(t *testing.T) {
	t.Run("VMI deletion fails", func(t *testing.T) {
		expectedVMIDeletionFailure := errors.New("failed to delete VMI")

		testClient := newClientStub()
		testClient.vmiDeletionFailure = expectedVMIDeletionFailure
		testCheckup := checkup.New(testClient, testNamespace, newTestConfig(), executorStub{})

		assert.NoError(t, testCheckup.Setup(context.Background()))
		assert.NoError(t, testCheckup.Run(context.Background()))

		assert.ErrorContains(t, testCheckup.Teardown(context.Background()), expectedVMIDeletionFailure.Error())
	})

	t.Run("wait for VMI deletion fails", func(t *testing.T) {
		expectedReadFailure := errors.New("failed to read VMI")

		testClient := newClientStub()
		testCheckup := checkup.New(testClient, testNamespace, newTestConfig(), executorStub{})

		assert.NoError(t, testCheckup.Setup(context.Background()))
		assert.NoError(t, testCheckup.Run(context.Background()))

		testClient.vmiReadFailure = expectedReadFailure
		assert.ErrorContains(t, testCheckup.Teardown(context.Background()), expectedReadFailure.Error())
	})
}

type clientStub struct {
	createdVMIs              map[string]*kvcorev1.VirtualMachineInstance
	vmiCreationFailure       error
	vmiReadFailure           error
	vmiDeletionFailure       error
	createdConfigMaps        map[string]*corev1.ConfigMap
	configMapCreationFailure error
}

func newClientStub() *clientStub {
	return &clientStub{
		createdVMIs:       map[string]*kvcorev1.VirtualMachineInstance{},
		createdConfigMaps: map[string]*corev1.ConfigMap{},
	}
}

func (cs *clientStub) CreateVirtualMachineInstance(_ context.Context,
	namespace string,
	vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	if cs.vmiCreationFailure != nil {
		return nil, cs.vmiCreationFailure
	}

	vmi.Namespace = namespace

	vmiFullName := checkup.ObjectFullName(vmi.Namespace, vmi.Name)
	cs.createdVMIs[vmiFullName] = vmi

	vmi.Status.Conditions = append(vmi.Status.Conditions, kvcorev1.VirtualMachineInstanceCondition{
		Type:   kvcorev1.VirtualMachineInstanceAgentConnected,
		Status: corev1.ConditionTrue,
	})

	return vmi, nil
}

func (cs *clientStub) GetVirtualMachineInstance(_ context.Context, namespace, name string) (*kvcorev1.VirtualMachineInstance, error) {
	if cs.vmiReadFailure != nil {
		return nil, cs.vmiReadFailure
	}

	vmiFullName := checkup.ObjectFullName(namespace, name)
	vmi, exist := cs.createdVMIs[vmiFullName]
	if !exist {
		return nil, k8serrors.NewNotFound(schema.GroupResource{Group: "kubevirt.io", Resource: "virtualmachineinstances"}, name)
	}

	return vmi, nil
}

func (cs *clientStub) DeleteVirtualMachineInstance(_ context.Context, namespace, name string) error {
	if cs.vmiDeletionFailure != nil {
		return cs.vmiDeletionFailure
	}

	vmiFullName := checkup.ObjectFullName(namespace, name)
	_, exist := cs.createdVMIs[vmiFullName]
	if !exist {
		return k8serrors.NewNotFound(schema.GroupResource{Group: "kubevirt.io", Resource: "virtualmachineinstances"}, name)
	}

	delete(cs.createdVMIs, vmiFullName)

	return nil
}

func (cs *clientStub) CreateConfigMap(_ context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	if cs.configMapCreationFailure != nil {
		return nil, cs.configMapCreationFailure
	}

	configMap.Namespace = namespace

	configMapFullName := checkup.ObjectFullName(configMap.Namespace, configMap.Name)
	cs.createdConfigMaps[configMapFullName] = configMap

	return configMap, nil
}

func (cs *clientStub) VMIName() string {
	for _, vmi := range cs.createdVMIs {
		if strings.Contains(vmi.Name, checkup.VMINamePrefix) {
			return vmi.Name
		}
	}

	return ""
}

func TestRunFailure(t *testing.T) {
	expectedExecutionFailure := errors.New("failed to execute realtime checkup")

	testClient := newClientStub()
	testConfig := newTestConfig()
	testCheckup := checkup.New(testClient, testNamespace, testConfig, executorStub{executeErr: expectedExecutionFailure})

	assert.NoError(t, testCheckup.Setup(context.Background()))

	vmiName := testClient.VMIName()
	assert.NotEmpty(t, vmiName)

	assert.Error(t, expectedExecutionFailure, testCheckup.Run(context.Background()))

	assert.NoError(t, testCheckup.Teardown(context.Background()))

	_, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, vmiName)
	assert.ErrorContains(t, err, "not found")

	actualResults := testCheckup.Results()
	expectedResults := status.Results{}

	assert.Equal(t, expectedResults, actualResults)
}

type executorStub struct {
	executeErr error
}

func (es executorStub) Execute(_ context.Context, vmiName string) (status.Results, error) {
	if es.executeErr != nil {
		return status.Results{}, es.executeErr
	}

	return status.Results{}, nil
}

func newTestConfig() config.Config {
	return config.Config{
		PodName:                   "",
		PodUID:                    "",
		VMUnderTestTargetNodeName: testTargetNodeName,
		OslatDuration:             10 * time.Minute,
		OslatLatencyThreshold:     45 * time.Microsecond,
	}
}
