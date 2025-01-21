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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package reporter

import (
	"fmt"
	"os"
	"path"

	"k8s.io/client-go/kubernetes"

	"github.com/kiagnose/kubevirt-realtime-checkup/tests/reporter/configmap"
	"github.com/kiagnose/kubevirt-realtime-checkup/tests/reporter/job"
)

type CheckupReporter struct {
	Client        *kubernetes.Clientset
	ArtifactsDir  string
	Namespace     string
	JobName       string
	ConfigMapName string
}

func New(params CheckupReporter) *CheckupReporter {
	return &CheckupReporter{
		Client:        params.Client,
		ArtifactsDir:  params.ArtifactsDir,
		Namespace:     params.Namespace,
		JobName:       params.JobName,
		ConfigMapName: params.ConfigMapName,
	}
}

// Cleanup cleans up the current content of the artifactsDir
func (c CheckupReporter) Cleanup() {
	// clean up artifacts from previous run
	if c.ArtifactsDir != "" {
		_, err := os.Stat(c.ArtifactsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return
			} else {
				panic(err)
			}
		}
		names, err := os.ReadDir(c.ArtifactsDir)
		if err != nil {
			panic(err)
		}
		for _, entry := range names {
			os.RemoveAll(path.Join([]string{c.ArtifactsDir, entry.Name()}...))
		}
	}
}

// CollectArtifacts collects the artifacts to artifactsDir
func (c CheckupReporter) CollectArtifacts() {
	const folderPermission = 0755
	err := os.MkdirAll(c.ArtifactsDir, folderPermission)
	if err != nil {
		fmt.Println(err)
		return
	}

	checkupJobLogs, err := job.GetLogs(c.Client, c.Namespace, c.JobName)
	if err != nil {
		fmt.Println(err)
		return
	}
	c.writeTestLogFile("job", c.JobName, checkupJobLogs)

	configmapData, err := configmap.GetData(c.Client, c.Namespace, c.ConfigMapName)
	if err != nil {
		fmt.Println(err)
		return
	}
	c.writeTestLogFile("configmap", c.ConfigMapName, configmapData)
}

func (c CheckupReporter) writeTestLogFile(objectType, objectName, logBuffer string) {
	const filePermission = 0644

	name := fmt.Sprintf("%s/%s_%s.log", c.ArtifactsDir, objectType, objectName)
	fi, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, filePermission)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if errFile := fi.Close(); errFile != nil {
			fmt.Println(errFile)
		}
	}()
	_, err = fi.WriteString(logBuffer)
	if err != nil {
		fmt.Println(err)
		return
	}
}
