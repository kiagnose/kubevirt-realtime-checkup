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

package job

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func GetLogs(client *kubernetes.Clientset, namespace, jobName string) (string, error) {
	jobPodName, err := getPodNameByJob(client, namespace, jobName)
	if err != nil {
		return "", err
	}

	rawLogs, err := client.CoreV1().Pods(namespace).GetLogs(jobPodName, &corev1.PodLogOptions{}).DoRaw(context.Background())
	if err != nil {
		return "", err
	}
	return string(rawLogs), nil
}

func getPodNameByJob(client *kubernetes.Clientset, namespace, jobName string) (string, error) {
	const JobNameLabel = "job-name"
	jobLabel := k8slabels.Set{JobNameLabel: jobName}
	podList, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: jobLabel.String()})
	if err != nil {
		return "", err
	}

	if len(podList.Items) != 1 {
		return "", fmt.Errorf("failed to find job's pod: found %d pods for Job %s/%s", len(podList.Items), namespace, jobName)
	}
	return podList.Items[0].Name, nil
}
