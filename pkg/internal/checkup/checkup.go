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
	"path"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8srand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/checkup/configmap"
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
	CreateConfigMap(ctx context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	DeleteConfigMap(ctx context.Context, namespace, name string) error
}

type testExecutor interface {
	Execute(ctx context.Context, vmiName string) (status.Results, error)
}

type Checkup struct {
	client               kubeVirtVMIClient
	namespace            string
	vmUnderTestConfigMap *corev1.ConfigMap
	vmi                  *kvcorev1.VirtualMachineInstance
	results              status.Results
	executor             testExecutor
	cfg                  config.Config
}

const (
	VMUnderTestConfigMapNamePrefix = "realtime-vm-config"
	VMINamePrefix                  = "realtime-vmi-under-test"
)

func New(client kubeVirtVMIClient, namespace string, checkupConfig config.Config, executor testExecutor) *Checkup {
	const randomStringLen = 5
	randomSuffix := k8srand.String(randomStringLen)

	vmiUnderTestCMName := vmiUnderTestConfigMapName(randomSuffix)

	return &Checkup{
		client:               client,
		namespace:            namespace,
		vmUnderTestConfigMap: newVMUnderTestConfigMap(vmiUnderTestCMName, checkupConfig),
		vmi:                  newRealtimeVMI(vmiUnderTestName(randomSuffix), checkupConfig, vmiUnderTestCMName),
		executor:             executor,
		cfg:                  checkupConfig,
	}
}

func (c *Checkup) Setup(ctx context.Context) error {
	const setupTimeout = 10 * time.Minute
	setupCtx, cancel := context.WithTimeout(ctx, setupTimeout)
	defer cancel()

	const errMessagePrefix = "Setup"

	if err := c.createVMUnderTestCM(setupCtx); err != nil {
		return fmt.Errorf("%s: %w", errMessagePrefix, err)
	}

	createdVMI, err := c.client.CreateVirtualMachineInstance(setupCtx, c.namespace, c.vmi)
	if err != nil {
		return err
	}
	c.vmi = createdVMI

	var updatedVMIUnderTest *kvcorev1.VirtualMachineInstance
	updatedVMIUnderTest, err = c.waitForVMIToBeReady(setupCtx)
	if err != nil {
		return err
	}

	c.vmi = updatedVMIUnderTest

	return nil
}

func (c *Checkup) Run(ctx context.Context) error {
	var err error

	c.results, err = c.executor.Execute(ctx, c.vmi.Name)
	if err != nil {
		return err
	}
	c.results.VMUnderTestActualNodeName = c.vmi.Status.NodeName

	if c.results.OslatMaxLatency > c.cfg.OslatLatencyThreshold {
		return fmt.Errorf("oslat Max Latency measured %s exceeded the given threshold %s",
			c.results.OslatMaxLatency.String(), c.cfg.OslatLatencyThreshold.String())
	}
	return nil
}

func (c *Checkup) Teardown(ctx context.Context) error {
	const errPrefix = "teardown"

	if err := c.deleteVMI(ctx); err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	if err := c.deleteVMUnderTestCM(ctx); err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	if err := c.waitForVMIDeletion(ctx); err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	return nil
}

func (c *Checkup) Results() status.Results {
	return c.results
}

func (c *Checkup) createVMUnderTestCM(ctx context.Context) error {
	log.Printf("Creating ConfigMap %q...", ObjectFullName(c.namespace, c.vmUnderTestConfigMap.Name))

	_, err := c.client.CreateConfigMap(ctx, c.namespace, c.vmUnderTestConfigMap)
	return err
}

func (c *Checkup) deleteVMUnderTestCM(ctx context.Context) error {
	log.Printf("Deleting ConfigMap %q...", ObjectFullName(c.namespace, c.vmUnderTestConfigMap.Name))

	return c.client.DeleteConfigMap(ctx, c.namespace, c.vmUnderTestConfigMap.Name)
}

func (c *Checkup) waitForVMIToBeReady(ctx context.Context) (*kvcorev1.VirtualMachineInstance, error) {
	vmiFullName := ObjectFullName(c.vmi.Namespace, c.vmi.Name)
	var updatedVMI *kvcorev1.VirtualMachineInstance

	log.Printf("Waiting for VMI %q to be ready...", vmiFullName)

	conditionFn := func(ctx context.Context) (bool, error) {
		var err error
		updatedVMI, err = c.client.GetVirtualMachineInstance(ctx, c.vmi.Namespace, c.vmi.Name)
		if err != nil {
			return false, err
		}

		for _, condition := range updatedVMI.Status.Conditions {
			if condition.Type == kvcorev1.VirtualMachineInstanceReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}
	const pollInterval = 5 * time.Second
	if err := wait.PollImmediateUntilWithContext(ctx, pollInterval, conditionFn); err != nil {
		return nil, fmt.Errorf("failed to wait for VMI %q be ready: %w", vmiFullName, err)
	}

	log.Printf("VMI %q has successfully reached ready condition", vmiFullName)

	return updatedVMI, nil
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

func newVMUnderTestConfigMap(name string, checkupConfig config.Config) *corev1.ConfigMap {
	vmUnderTestConfigData := map[string]string{
		config.BootScriptName: generateBootScript(),
	}
	return configmap.New(name,
		checkupConfig.PodName,
		checkupConfig.PodUID,
		vmUnderTestConfigData)
}

func newRealtimeVMI(name string, checkupConfig config.Config, configMapName string) *kvcorev1.VirtualMachineInstance {
	const (
		CPUSocketsCount   = 1
		CPUCoresCount     = 4
		CPUTreadsCount    = 1
		hugePageSize      = "1Gi"
		guestMemory       = "4Gi"
		rootDiskName      = "rootdisk"
		configDiskSerial  = "DEADBEEF"
		cloudInitDiskName = "cloudinitdisk"
		configVolumeName  = "realtime-config"
	)

	return vmi.New(name,
		vmi.WithOwnerReference(checkupConfig.PodName, checkupConfig.PodUID),
		vmi.WithoutCRIOCPULoadBalancing(),
		vmi.WithoutCRIOCPUQuota(),
		vmi.WithoutCRIOIRQLoadBalancing(),
		vmi.WithRealtimeCPU(CPUSocketsCount, CPUCoresCount, CPUTreadsCount),
		vmi.WithMemory(hugePageSize, guestMemory),
		vmi.WithoutAutoAttachGraphicsDevice(),
		vmi.WithoutAutoAttachMemBalloon(),
		vmi.WithAutoAttachSerialConsole(),
		vmi.WithZeroTerminationGracePeriodSeconds(),
		vmi.WithNodeSelector(checkupConfig.VMUnderTestTargetNodeName),
		vmi.WithContainerDisk(rootDiskName, checkupConfig.VMUnderTestContainerDiskImage),
		vmi.WithVirtIODisk(rootDiskName),
		vmi.WithConfigMapVolume(configVolumeName, configMapName),
		vmi.WithConfigMapDisk(configVolumeName, configDiskSerial),
		vmi.WithCloudInitNoCloudVolume(cloudInitDiskName,
			vmi.CloudInit(realtimeVMIBootCommands(configDiskSerial))),
		vmi.WithReadinessFileProbe(config.BootScriptReadinessMarkerFileFullPath),
	)
}

func generateBootScript() string {
	const isolatedCores = "2-3"
	sb := strings.Builder{}

	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("set -x\n")
	sb.WriteString("\n")
	sb.WriteString("checkup_tuned_adm_set_marker_full_path=" + config.BootScriptTunedAdmSetMarkerFileFullPath + "\n")
	sb.WriteString("\n")
	sb.WriteString("if systemctl --type swap list-units | grep -q '.swap'; then\n")
	sb.WriteString("  systemctl mask \"$(systemctl --type swap list-units | grep '.swap' | awk '{print $1}')\"\n")
	sb.WriteString("fi\n")
	sb.WriteString("\n")
	sb.WriteString("if [ ! -f \"$checkup_tuned_adm_set_marker_full_path\" ]; then\n")
	sb.WriteString("  tuned_conf=\"/etc/tuned/realtime-virtual-guest-variables.conf\"\n")
	sb.WriteString("  echo \"isolated_cores=" + isolatedCores + "\" > \"$tuned_conf\"\n")
	sb.WriteString("  echo \"isolate_managed_irq=Y\" >> \"$tuned_conf\"\n")
	sb.WriteString("  systemctl restart tuned.service\n")
	sb.WriteString("  tuned-adm profile realtime-virtual-guest\n")
	sb.WriteString("  touch $checkup_tuned_adm_set_marker_full_path\n")
	sb.WriteString("  reboot\n")
	sb.WriteString("  exit 0\n")
	sb.WriteString("fi\n")
	sb.WriteString("\n")
	sb.WriteString("touch " + config.BootScriptReadinessMarkerFileFullPath + "\n")
	sb.WriteString("chcon -t virt_qemu_ga_exec_t " + config.BootScriptReadinessMarkerFileFullPath + "\n")

	return sb.String()
}

func realtimeVMIBootCommands(configDiskSerial string) []string {
	const configMountDirectory = "/mnt/app-config"

	return []string{
		fmt.Sprintf("mkdir %s", configMountDirectory),
		fmt.Sprintf("mount /dev/$(lsblk --nodeps -no name,serial | grep %s | cut -f1 -d' ') %s", configDiskSerial, configMountDirectory),
		fmt.Sprintf("cp %s %s", path.Join(configMountDirectory, config.BootScriptName), config.BootScriptBinDirectory),
		fmt.Sprintf("chmod 744 %s", path.Join(config.BootScriptBinDirectory, config.BootScriptName)),
		path.Join(config.BootScriptBinDirectory, config.BootScriptName),
	}
}

func vmiUnderTestName(suffix string) string {
	return VMINamePrefix + "-" + suffix
}

func vmiUnderTestConfigMapName(suffix string) string {
	return VMUnderTestConfigMapNamePrefix + "-" + suffix
}

func ObjectFullName(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
