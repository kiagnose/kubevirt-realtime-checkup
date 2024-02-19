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

package configmap

import (
	k8scorev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func New(name, ownerName, ownerUID string) *k8scorev1.ConfigMap {
	return &k8scorev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       ownerName,
					UID:        types.UID(ownerUID),
				},
			},
		},
	}
}
