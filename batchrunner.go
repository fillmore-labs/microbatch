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

package microbatch

import (
	"time"
)

type batchRunner[Q, S any, K comparable, QQ ~[]Q, SS ~[]S] struct {
	batchSize     int
	batchDuration time.Duration
	requestChan   <-chan batchRequest[Q, S]
	processor     *processor[Q, S, K, QQ, SS]
	timerRunning  bool
	timer         *time.Timer
	batch         []batchRequest[Q, S]
}

// Run batch collection until the request channel is cloes.
func (b *batchRunner[Q, S, K, QQ, SS]) runBatcher() {
	b.timer = newTimer()
	b.batch = make([]batchRequest[Q, S], 0, b.batchSize)

	for {
		select {
		case request, ok := <-b.requestChan:
			if !ok {
				b.sendBatch()

				return
			}

			b.batch = append(b.batch, request)

			switch len(b.batch) {
			case b.batchSize:
				b.sendBatch()

			case 1:
				// Start timer if this is the first entry in the batch
				b.timerRunning = true
				b.timer.Reset(b.batchDuration)
			}

		case <-b.timer.C:
			// Send out batch
			b.timerRunning = false
			b.sendBatch()
		}
	}
}

// Stop timer and send out batch data.
func (b *batchRunner[Q, S, K, QQ, SS]) sendBatch() {
	if b.timerRunning {
		if !b.timer.Stop() {
			<-b.timer.C
		}
		b.timerRunning = false
	}

	go b.processor.process(b.batch)
	b.batch = make([]batchRequest[Q, S], 0, b.batchSize)
}

func newTimer() *time.Timer {
	timer := time.NewTimer(0)
	<-timer.C

	return timer
}
