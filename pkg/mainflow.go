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

package pkg

import (
	"context"
	"log"

	kconfig "github.com/kiagnose/kiagnose/kiagnose/config"

	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/checkup"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/client"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/config"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/launcher"
	"github.com/kiagnose/kubevirt-realtime-checkup/pkg/internal/reporter"
)

func Run(rawEnv map[string]string, namespace string) error {
	c, err := client.New()
	if err != nil {
		return err
	}

	baseConfig, err := kconfig.Read(c, rawEnv)
	if err != nil {
		return err
	}

	cfg, err := config.New(baseConfig)
	if err != nil {
		return err
	}

	printConfig(cfg)

	l := launcher.New(checkup.New(c, namespace, cfg), reporter.New(c, baseConfig.ConfigMapNamespace, baseConfig.ConfigMapName))

	ctx, cancel := context.WithTimeout(context.Background(), baseConfig.Timeout)
	defer cancel()

	return l.Run(ctx)
}

func printConfig(checkupConfig config.Config) {
	log.Println("Using the following config:")
	log.Printf("\t%q: %q", config.VMUnderTestTargetNodeNameParamName, checkupConfig.VMUnderTestTargetNodeName)
	log.Printf("\t%q: %q", config.VMUnderTestContainerDiskImageParamName, checkupConfig.VMUnderTestContainerDiskImage)
	log.Printf("\t%q: %q", config.OslatDurationParamName, checkupConfig.OslatDuration.String())
	log.Printf("\t%q: %q", config.OslatLatencyThresholdParamName, checkupConfig.OslatLatencyThreshold.String())
}
