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
	"os"
	"path"
)

type CheckupReporter struct {
	ArtifactsDir string
}

func New(params CheckupReporter) *CheckupReporter {
	return &CheckupReporter{
		ArtifactsDir: params.ArtifactsDir,
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
