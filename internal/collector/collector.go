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

	internal "fillmore-labs.com/microbatch/internal/types"
)

// Processor defines the interface for processing batches of requests.
type Processor[Q, S any] interface {
	Process(requests []internal.BatchRequest[Q, S])
}

// Collector handles batch collection and processing of requests.
//
// Collects requests until the batch size or duration is reached, then sends them to the Processor.
type Collector[Q, S any] struct {
	// Requests is the channel for receiving new batch requests.
	Requests <-chan internal.BatchRequest[Q, S]
	// Terminating is a signal for the Collector to shut down.
	Terminating <-chan struct{}
	// Terminated is closed after the final batch is processed on shutdown.
	Terminated chan<- struct{}

	// Processor processes batches of requests.
	Processor Processor[Q, S]

	// BatchSize is the maximum number of requests per batch or zero, when unlimited.
	BatchSize int
	// BatchDuration is the maximum time a batch can collect before processing or zero, when unlimited.
	BatchDuration time.Duration

	// Timer tracks the batch duration and signals when it expires.
	Timer        *Timer
	timerRunning bool

	// batch holds the collected requests until processing.
	batch []internal.BatchRequest[Q, S]
}

// Run runs the main collection loop.
func (c *Collector[Q, S]) Run() {
	c.init()

CollectorLoop:
	for {
		select {
		case request := <-c.Requests: // New request.
			c.addRequest(request)

		case <-c.Timer.C: // Batch timer expired.
			c.timerRunning = false
			c.sendBatch()

		case <-c.Terminating: // Shut down.
			c.stopTimer()
			if len(c.batch) > 0 {
				c.sendBatch()
			}
			close(c.Terminated)

			break CollectorLoop
		}
	}
}

// init sets up the Collector.
func (c *Collector[Q, S]) init() {
	if c.Timer == nil {
		if c.BatchDuration > 0 {
			c.Timer = NewTimer()
		} else {
			c.Timer = &Timer{}
		}
	}
	c.batch = c.newBatch()
}

// addRequest adds the given request to the batch. If the batch is full, send it out and start new batch.
func (c *Collector[Q, S]) addRequest(request internal.BatchRequest[Q, S]) {
	c.batch = append(c.batch, request)

	switch len(c.batch) {
	case c.BatchSize: // Batch full.
		c.stopTimer()
		c.sendBatch()

	case 1: // Start the timer if this is the first entry in the batch.
		c.startTimer()
	}
}

func (c *Collector[Q, S]) startTimer() {
	if c.BatchDuration > 0 {
		c.Timer.Reset(c.BatchDuration)
		c.timerRunning = true
	}
}

func (c *Collector[Q, S]) stopTimer() {
	if c.timerRunning {
		if !c.Timer.Stop() {
			<-c.Timer.C
		}
		c.timerRunning = false
	}
}

// Send the current batch to the processor and reset for new batch.
func (c *Collector[Q, S]) sendBatch() {
	go c.Processor.Process(c.batch)
	c.batch = c.newBatch()
}

func (c *Collector[Q, S]) newBatch() []internal.BatchRequest[Q, S] {
	if c.BatchSize > 0 {
		return make([]internal.BatchRequest[Q, S], 0, c.BatchSize)
	}

	return nil
}
