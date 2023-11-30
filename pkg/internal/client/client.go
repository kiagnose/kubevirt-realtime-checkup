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

package client

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	kvcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

type Client struct {
	kubecli.KubevirtClient
}

func New() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubecli.GetKubevirtClientFromRESTConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{client}, nil
}

func (c *Client) CreateVirtualMachineInstance(ctx context.Context,
	namespace string,
	vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	return c.KubevirtClient.VirtualMachineInstance(namespace).Create(ctx, vmi)
}

func (c *Client) GetVirtualMachineInstance(ctx context.Context, namespace, name string) (*kvcorev1.VirtualMachineInstance, error) {
	return c.KubevirtClient.VirtualMachineInstance(namespace).Get(ctx, name, &metav1.GetOptions{})
}

func (c *Client) DeleteVirtualMachineInstance(ctx context.Context, namespace, name string) error {
	return c.KubevirtClient.VirtualMachineInstance(namespace).Delete(ctx, name, &metav1.DeleteOptions{})
}

func (c *Client) VMISerialConsole(namespace, name string, timeout time.Duration) (kubecli.StreamInterface, error) {
	return c.KubevirtClient.VirtualMachineInstance(namespace).SerialConsole(
		name,
		&kubecli.SerialConsoleOptions{ConnectionTimeout: timeout},
	)
}
