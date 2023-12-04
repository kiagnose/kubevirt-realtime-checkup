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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8srand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/checkup/vmi"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/config"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/status"
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
	const setupTimeout = 10 * time.Minute
	setupCtx, cancel := context.WithTimeout(ctx, setupTimeout)
	defer cancel()

	createdVMI, err := c.client.CreateVirtualMachineInstance(setupCtx, c.namespace, c.vmi)
	if err != nil {
		return err
	}
	c.vmi = createdVMI

	if err := c.waitForVMIToBoot(setupCtx); err != nil {
		return err
	}

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

func (c *Checkup) waitForVMIToBoot(ctx context.Context) error {
	vmiFullName := ObjectFullName(c.vmi.Namespace, c.vmi.Name)
	log.Printf("Waiting for VMI %q to boot...", vmiFullName)

	conditionFn := func(ctx context.Context) (bool, error) {
		fetchedVMI, err := c.client.GetVirtualMachineInstance(ctx, c.vmi.Namespace, c.vmi.Name)
		if err != nil {
			return false, err
		}

		for _, condition := range fetchedVMI.Status.Conditions {
			if condition.Type == kvcorev1.VirtualMachineInstanceAgentConnected && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}
	const pollInterval = 5 * time.Second
	if err := wait.PollImmediateUntilWithContext(ctx, pollInterval, conditionFn); err != nil {
		return fmt.Errorf("failed to wait for VMI %q to boot: %w", vmiFullName, err)
	}

	log.Printf("VMI %q had successfully booted", vmiFullName)

	return nil
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
		CPUSocketsCount   = 1
		CPUCoresCount     = 3
		CPUTreadsCount    = 1
		hugePageSize      = "1Gi"
		guestMemory       = "4Gi"
		rootDiskName      = "rootdisk"
		cloudInitDiskName = "cloudinitdisk"
	)

	return vmi.New(randomizeName(VMINamePrefix),
		vmi.WithoutCRIOCPULoadBalancing(),
		vmi.WithoutCRIOCPUQuota(),
		vmi.WithoutCRIOIRQLoadBalancing(),
		vmi.WithRealtimeCPU(CPUSocketsCount, CPUCoresCount, CPUTreadsCount),
		vmi.WithMemory(hugePageSize, guestMemory),
		vmi.WithoutAutoAttachGraphicsDevice(),
		vmi.WithoutAutoAttachMemBalloon(),
		vmi.WithAutoAttachSerialConsole(),
		vmi.AutoIOThreadPolicy(),
		vmi.WithZeroTerminationGracePeriodSeconds(),
		vmi.WithNodeSelector(checkupConfig.VMUnderTestTargetNodeName),
		vmi.WithContainerDisk(rootDiskName, checkupConfig.VMUnderTestContainerDiskImage),
		vmi.WithVirtIODisk(rootDiskName),
		vmi.WithCloudInitNoCloudVolume(cloudInitDiskName, CloudInit(config.VMIUsername, config.VMIPassword)),
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

func CloudInit(username, password string) string {
	sb := strings.Builder{}
	sb.WriteString("#cloud-config\n")
	sb.WriteString(fmt.Sprintf("user: %s\n", username))
	sb.WriteString(fmt.Sprintf("password: %s\n", password))
	sb.WriteString("chpasswd:\n")
	sb.WriteString("  expire: false")

	return sb.String()
}
