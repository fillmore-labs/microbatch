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

package timer

import "time"

// Timer is an interface to control a Timer.
type Timer interface {
	Stop()
}

type timer struct {
	timer *time.Timer
	sent  *bool
}

func (t timer) Stop() {
	if !t.timer.Stop() {
		*t.sent = true
	}
}

func New(d time.Duration, f func(sent *bool)) Timer {
	sent := new(bool)

	return timer{
		timer: time.AfterFunc(d, func() { f(sent) }),
		sent:  sent,
	}
}
