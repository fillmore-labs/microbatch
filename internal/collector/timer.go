// Copyright 2023-2024 Oliver Eikemeier. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"time"
)

type Timer struct {
	C     <-chan time.Time
	Timer TimerDelegate
}

type TimerDelegate interface {
	Stop() bool
	Reset(d time.Duration) bool
}

// NewTimer creates a new timer that is not running.
func NewTimer() *Timer {
	t := time.NewTimer(time.Hour)
	if !t.Stop() {
		<-t.C
	}

	return &Timer{C: t.C, Timer: t}
}

func (t *Timer) Stop() bool {
	return t.Timer.Stop()
}

func (t *Timer) Reset(d time.Duration) bool {
	return t.Timer.Reset(d)
}
