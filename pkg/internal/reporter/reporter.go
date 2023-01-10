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

package reporter

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	kreporter "github.com/kiagnose/kiagnose/kiagnose/reporter"

	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/status"
)

const (
	NodeKey            = "node"
	OslatMaxLatencyKey = "oslatMaxLatencyMicroSeconds"
)

type Reporter struct {
	kreporter.Reporter
}

func New(c kubernetes.Interface, configMapNamespace, configMapName string) *Reporter {
	r := kreporter.New(c, configMapNamespace, configMapName)
	return &Reporter{*r}
}

func (r *Reporter) Report(checkupStatus status.Status) error {
	if !r.HasData() {
		return r.Reporter.Report(checkupStatus.Status)
	}

	checkupStatus.Succeeded = len(checkupStatus.FailureReason) == 0

	checkupStatus.Status.Results = formatResults(checkupStatus)

	return r.Reporter.Report(checkupStatus.Status)
}

func formatResults(checkupStatus status.Status) map[string]string {
	var emptyResults status.Results
	if checkupStatus.Results == emptyResults {
		return map[string]string{}
	}

	formattedResults := map[string]string{
		NodeKey:            checkupStatus.Results.Node,
		OslatMaxLatencyKey: fmt.Sprintf("%d", checkupStatus.Results.OslatMaxLatency.Microseconds()),
	}

	return formattedResults
}
