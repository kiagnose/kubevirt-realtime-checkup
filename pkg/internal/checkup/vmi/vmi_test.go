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

package vmi_test

import (
	"testing"

	assert "github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/checkup/vmi"
)

const testVMIName = "my-vm"

func TestNew(t *testing.T) {
	actualVMI := vmi.New(testVMIName)
	expectedVMI := newBaseVMI()

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithOwnerReference(t *testing.T) {
	const (
		testPodName = "rt-checkup-1234"
		testPodUID  = "0123456789-0123456789"
	)

	actualVMI := vmi.New(testVMIName, vmi.WithOwnerReference(testPodName, testPodUID))

	expectedVMI := newBaseVMI()
	expectedVMI.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       testPodName,
			UID:        testPodUID,
		},
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithoutCRIOCPULoadBalancing(t *testing.T) {
	actualVMI := vmi.New(testVMIName, vmi.WithoutCRIOCPULoadBalancing())

	expectedVMI := newBaseVMI()
	expectedVMI.ObjectMeta.Annotations = map[string]string{
		vmi.CRIOCPULoadBalancingAnnotation: vmi.Disable,
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithoutCRIOCPUQuota(t *testing.T) {
	actualVMI := vmi.New(testVMIName, vmi.WithoutCRIOCPUQuota())

	expectedVMI := newBaseVMI()
	expectedVMI.ObjectMeta.Annotations = map[string]string{
		vmi.CRIOCPUQuotaAnnotation: vmi.Disable,
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithoutCRIOIRQLoadBalancing(t *testing.T) {
	actualVMI := vmi.New(testVMIName, vmi.WithoutCRIOIRQLoadBalancing())

	expectedVMI := newBaseVMI()
	expectedVMI.ObjectMeta.Annotations = map[string]string{
		vmi.CRIOIRQLoadBalancingAnnotation: vmi.Disable,
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithRealtimeCPU(t *testing.T) {
	actualVMI := vmi.New(testVMIName, vmi.WithRealtimeCPU())

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.Domain.CPU = &kvcorev1.CPU{
		Model:                 "host-passthrough",
		DedicatedCPUPlacement: true,
		IsolateEmulatorThread: true,

		Features: []kvcorev1.CPUFeature{
			{Name: "tsc-deadline", Policy: "require"},
		},
		NUMA: &kvcorev1.NUMA{
			GuestMappingPassthrough: &kvcorev1.NUMAGuestMappingPassthrough{},
		},
		Realtime: &kvcorev1.Realtime{Mask: "1-2"},
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithoutAutoattachGraphicsDevice(t *testing.T) {
	actualVMI := vmi.New(testVMIName, vmi.WithoutAutoAttachGraphicsDevice())

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.Domain.Devices.AutoattachGraphicsDevice = vmi.Pointer(false)

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithoutAutoattachMemBalloon(t *testing.T) {
	actualVMI := vmi.New(testVMIName, vmi.WithoutAutoAttachMemBalloon())

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.Domain.Devices.AutoattachMemBalloon = vmi.Pointer(false)

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithAutoattachSerialConsole(t *testing.T) {
	actualVMI := vmi.New(testVMIName, vmi.WithAutoAttachSerialConsole())

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.Domain.Devices.AutoattachSerialConsole = vmi.Pointer(true)

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithVirtIODisk(t *testing.T) {
	const disk1Name = "rootdisk"
	const disk2Name = "my-disk2"

	actualVMI := vmi.New(testVMIName,
		vmi.WithVirtIODisk(disk1Name),
		vmi.WithVirtIODisk(disk2Name),
	)

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.Domain.Devices.Disks = []kvcorev1.Disk{
		{
			Name: disk1Name,
			DiskDevice: kvcorev1.DiskDevice{
				Disk: &kvcorev1.DiskTarget{Bus: kvcorev1.DiskBusVirtio},
			},
		},
		{
			Name: disk2Name,
			DiskDevice: kvcorev1.DiskDevice{
				Disk: &kvcorev1.DiskTarget{Bus: kvcorev1.DiskBusVirtio},
			},
		},
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithHugePages(t *testing.T) {
	actualVMI := vmi.New(testVMIName, vmi.WithHugePages())

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.Domain.Memory = &kvcorev1.Memory{
		Hugepages: &kvcorev1.Hugepages{PageSize: "1Gi"},
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithResources(t *testing.T) {
	const (
		cpu    = "3"
		memory = "8Gi"
	)
	actualVMI := vmi.New(testVMIName, vmi.WithResources(cpu, memory))

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.Domain.Resources = kvcorev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(memory),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(memory),
		},
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithNodeSelectorWhen(t *testing.T) {
	t.Run("hostname is not empty", func(t *testing.T) {
		const nodeName = "my-node"
		actualVMI := vmi.New(testVMIName, vmi.WithNodeSelector(nodeName))

		expectedVMI := newBaseVMI()
		expectedVMI.Spec.NodeSelector = map[string]string{
			corev1.LabelHostname: nodeName,
		}

		assert.Equal(t, expectedVMI, actualVMI)
	})

	t.Run("hostname is empty", func(t *testing.T) {
		actualVMI := vmi.New(testVMIName, vmi.WithNodeSelector(""))

		expectedVMI := newBaseVMI()
		expectedVMI.Spec.NodeSelector = nil

		assert.Equal(t, expectedVMI, actualVMI)
	})
}

func TestNewWithTerminationGracePeriodSeconds(t *testing.T) {
	actualVMI := vmi.New(testVMIName, vmi.WithZeroTerminationGracePeriodSeconds())

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.TerminationGracePeriodSeconds = vmi.Pointer(int64(0))

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithPVCVolume(t *testing.T) {
	const (
		volumeName = "rootdisk"
		pvcName    = "centos-8-rt"
	)

	actualVMI := vmi.New(testVMIName, vmi.WithPVCVolume(volumeName, pvcName))

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.Volumes = []kvcorev1.Volume{
		{
			Name: volumeName,
			VolumeSource: kvcorev1.VolumeSource{
				PersistentVolumeClaim: &kvcorev1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			},
		},
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func TestNewWithCloudInitNoCloudVolume(t *testing.T) {
	const (
		volumeName = "rootdisk"
		userData   = `#cloud-config
password: redhat
chpasswd:
  expire: false
user: user`
	)

	actualVMI := vmi.New(testVMIName, vmi.WithCloudInitNoCloudVolume(volumeName, userData))

	expectedVMI := newBaseVMI()
	expectedVMI.Spec.Volumes = []kvcorev1.Volume{
		{
			Name: volumeName,
			VolumeSource: kvcorev1.VolumeSource{
				CloudInitNoCloud: &kvcorev1.CloudInitNoCloudSource{
					UserData: userData,
				},
			},
		},
	}

	assert.Equal(t, expectedVMI, actualVMI)
}

func newBaseVMI() *kvcorev1.VirtualMachineInstance {
	return &kvcorev1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			Kind:       kvcorev1.VirtualMachineInstanceGroupVersionKind.Kind,
			APIVersion: kvcorev1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: testVMIName,
		},
	}
}
