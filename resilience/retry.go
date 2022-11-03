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

// Package resilience provides helpers for dealing with resilience.
package resilience

import (
	"time"

	"github.com/pkg/errors"

	"github.com/ory/x/logrusx"
)

// Retry executes a f until no error is returned or failAfter is reached.
func Retry(logger *logrusx.Logger, maxWait time.Duration, failAfter time.Duration, f func() error) (err error) {
	var lastStart time.Time
	err = errors.New("did not connect")
	loopWait := time.Millisecond * 100
	retryStart := time.Now().UTC()
	for retryStart.Add(failAfter).After(time.Now().UTC()) {
		lastStart = time.Now().UTC()
		if err = f(); err == nil {
			return nil
		}

		if lastStart.Add(maxWait * 2).Before(time.Now().UTC()) {
			retryStart = time.Now().UTC()
		}

		logger.WithError(err).Infof("Retrying in %f seconds...", loopWait.Seconds())
		time.Sleep(loopWait)
		loopWait = loopWait * time.Duration(int64(2))
		if loopWait > maxWait {
			loopWait = maxWait
		}
	}
	return err
}
