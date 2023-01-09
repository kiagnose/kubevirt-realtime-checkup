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

package reporter_test

import (
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/reporter"
	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/status"
)

func TestReportShouldSucceed(t *testing.T) {
	testReporter := reporter.New()

	assert.NoError(t, testReporter.Report(status.Status{}))
}
