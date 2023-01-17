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

package checkup

import (
	"context"
	"fmt"
	"log"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8srand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/checkup/vmi"
	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/config"
	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/status"
)

type kubeVirtVMIClient interface {
	CreateVirtualMachineInstance(ctx context.Context,
		namespace string,
		vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error)
	GetVirtualMachineInstance(ctx context.Context, namespace, name string) (*kvcorev1.VirtualMachineInstance, error)
	DeleteVirtualMachineInstance(ctx context.Context, namespace, name string) error
}

type Checkup struct {
	client    kubeVirtVMIClient
	namespace string
	vmi       *kvcorev1.VirtualMachineInstance
}

const VMINamePrefix = "rt-vmi"

func New(client kubeVirtVMIClient, namespace string, checkupConfig config.Config) *Checkup {
	return &Checkup{
		client:    client,
		namespace: namespace,
		vmi:       newRealtimeVMI(checkupConfig),
	}
}

func (c *Checkup) Setup(ctx context.Context) error {
	createdVMI, err := c.client.CreateVirtualMachineInstance(ctx, c.namespace, c.vmi)
	if err != nil {
		return err
	}
	c.vmi = createdVMI

	return nil
}

func (c *Checkup) Run(ctx context.Context) error {
	return nil
}

func (c *Checkup) Teardown(ctx context.Context) error {
	const errPrefix = "teardown"

	if err := c.deleteVMI(ctx); err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	if err := c.waitForVMIDeletion(ctx); err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	return nil
}

func (c *Checkup) Results() status.Results {
	return status.Results{}
}

func (c *Checkup) deleteVMI(ctx context.Context) error {
	if c.vmi == nil {
		return fmt.Errorf("failed to delete VMI, object doesn't exist")
	}

	vmiFullName := ObjectFullName(c.vmi.Namespace, c.vmi.Name)

	log.Printf("Trying to delete VMI: %q", vmiFullName)
	if err := c.client.DeleteVirtualMachineInstance(ctx, c.vmi.Namespace, c.vmi.Name); err != nil {
		log.Printf("Failed to delete VMI: %q", vmiFullName)
		return err
	}

	return nil
}

func (c *Checkup) waitForVMIDeletion(ctx context.Context) error {
	vmiFullName := ObjectFullName(c.vmi.Namespace, c.vmi.Name)
	log.Printf("Waiting for VMI %q to be deleted...", vmiFullName)

	conditionFn := func(ctx context.Context) (bool, error) {
		_, err := c.client.GetVirtualMachineInstance(ctx, c.vmi.Namespace, c.vmi.Name)
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	const pollInterval = 5 * time.Second
	if err := wait.PollImmediateUntilWithContext(ctx, pollInterval, conditionFn); err != nil {
		return fmt.Errorf("failed to wait for VMI %q to be deleted: %v", vmiFullName, err)
	}

	log.Printf("VMI %q was deleted successfully", vmiFullName)

	return nil
}

func newRealtimeVMI(checkupConfig config.Config) *kvcorev1.VirtualMachineInstance {
	const (
		rootDiskName      = "rootdisk"
		cloudInitDiskName = "cloudinitdisk"
		userData          = `#cloud-config
password: redhat
chpasswd:
  expire: false
user: user`
	)

	return vmi.New(randomizeName(VMINamePrefix),
		vmi.WithoutCRIOCPULoadBalancing(),
		vmi.WithoutCRIOCPUQuota(),
		vmi.WithoutCRIOIRQLoadBalancing(),
		vmi.WithRealtimeCPU(),
		vmi.WithoutAutoAttachGraphicsDevice(),
		vmi.WithoutAutoAttachMemBalloon(),
		vmi.WithAutoAttachSerialConsole(),
		vmi.WithHugePages(),
		vmi.WithResources("3", "8Gi"),
		vmi.WithZeroTerminationGracePeriodSeconds(),
		vmi.WithNodeSelector(checkupConfig.TargetNode),
		vmi.WithPVCVolume(rootDiskName, checkupConfig.GuestImageSourcePVCName),
		vmi.WithVirtIODisk(rootDiskName),
		vmi.WithCloudInitNoCloudVolume(cloudInitDiskName, userData),
		vmi.WithVirtIODisk(cloudInitDiskName),
	)
}

func randomizeName(prefix string) string {
	const randomStringLen = 5

	return fmt.Sprintf("%s-%s", prefix, k8srand.String(randomStringLen))
}

func ObjectFullName(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
