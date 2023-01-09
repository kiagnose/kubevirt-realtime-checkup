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

package checkup_test

import (
	"context"
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/checkup"
	"github.com/kiagnose/kubevirt-rt-checkup/pkg/internal/status"
)

func TestCheckupShouldSucceed(t *testing.T) {
	testCheckup := checkup.New()

	assert.NoError(t, testCheckup.Setup(context.Background()))
	assert.NoError(t, testCheckup.Run(context.Background()))
	assert.NoError(t, testCheckup.Teardown(context.Background()))

	actualResults := testCheckup.Results()
	expectedResults := status.Results{}

	assert.Equal(t, expectedResults, actualResults)
}
