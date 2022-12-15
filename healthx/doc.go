// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

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
