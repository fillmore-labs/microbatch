// Copyright 2023 Oliver Eikemeier. All Rights Reserved.
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

package microbatch

import (
	"time"
)

type batchRunner[Q, S any] struct {
	requests      <-chan batchRequest[Q, S]
	batchSize     int
	batchDuration time.Duration
	processor     processor[Q, S]
	timerRunning  bool
	timer         *time.Timer
	batch         []batchRequest[Q, S]
}

// Run batch collection until the request channel is closed.
func (b *batchRunner[Q, S]) runBatcher() {
	b.timer = newTimer()
	b.batch = b.newBatch()

	for {
		select {
		case request, ok := <-b.requests:
			if !ok {
				b.sendBatch()

				return
			}

			b.addRequest(request)

		case <-b.timer.C:
			// Send out batch
			b.timerRunning = false
			b.sendBatch()
		}
	}
}

// Add a request to the batch. If it is full, send it out.
func (b *batchRunner[Q, S]) addRequest(request batchRequest[Q, S]) {
	b.batch = append(b.batch, request)

	switch len(b.batch) {
	case b.batchSize:
		b.sendBatch()

	case 1:
		if b.batchDuration > 0 {
			// Start timer if this is the first entry in the batch
			b.timer.Reset(b.batchDuration)
			b.timerRunning = true
		}
	}
}

// Stop timer and send out batch data.
func (b *batchRunner[Q, S]) sendBatch() {
	if b.timerRunning {
		if !b.timer.Stop() {
			<-b.timer.C
		}
		b.timerRunning = false
	}

	go b.processor.process(b.batch)
	b.batch = b.newBatch()
}

func (b *batchRunner[Q, S]) newBatch() []batchRequest[Q, S] {
	if b.batchSize > 0 {
		return make([]batchRequest[Q, S], 0, b.batchSize)
	}

	return nil
}

// Creates a new timer that is not running.
func newTimer() *time.Timer {
	timer := time.NewTimer(0)
	<-timer.C

	return timer
}
