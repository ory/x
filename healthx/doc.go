// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright © 2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
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
 * @author		Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @copyright 	2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 */

// Package healthx providers helpers for returning health status information via HTTP.
package healthx

import "strings"

// swagger:model healthStatus
type swaggerHealthStatus struct {
	// Status always contains "ok".
	Status string `json:"status"`
}

// swagger:model healthNotReadyStatus
type swaggerNotReadyStatus struct {
	// Errors contains a list of errors that caused the not ready status.
	Errors map[string]string `json:"errors"`
}

func (s swaggerNotReadyStatus) Error() string {
	var errs []string
	for _, err := range s.Errors {
		errs = append(errs, err)
	}
	return strings.Join(errs, "; ")
}

// swagger:model version
type swaggerVersion struct {
	// Version is the service's version.
	Version string `json:"version"`
}
