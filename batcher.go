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

package microbatch

import (
	"context"
	"errors"
	"time"

	"fillmore-labs.com/microbatch/internal/processor"
	"fillmore-labs.com/microbatch/internal/timer"
	internal "fillmore-labs.com/microbatch/internal/types"
	"fillmore-labs.com/promise"
)

// Batcher handles submitting requests in batches and returning results through channels.
type Batcher[Q, R any] struct {
	// process processes batches of requests.
	process func(requests []internal.BatchRequest[Q, R])

	// queue holds the collected requests until processing.
	queue chan []internal.BatchRequest[Q, R]

	// batchSize is the maximum number of requests per batch or zero, when unlimited.
	batchSize int

	// batchDuration is the maximum time a batch can collect before processing or zero, when unlimited.
	batchDuration time.Duration

	// timer tracks the batch duration and signals when it expires.
	timer timer.Timer

	// newTimer creates a new timer or a mock for testing
	newTimer func(d time.Duration, f func(sent *bool)) timer.Timer
}

var (
	// ErrNoResult is returned when the response from processJobs is missing a
	// matching correlation ID.
	ErrNoResult = errors.New("no result")
	// ErrDuplicateID is returned when a job has an already existing correlation ID.
	ErrDuplicateID = errors.New("duplicate correlation ID")
)

// NewBatcher creates a new [Batcher].
//
//   - batchProcessor is used to process batches of jobs.
//   - correlateRequest and correlateResult functions are used to get a common key from a job and result for
//     correlating results back to jobs.
//   - opts are used to configure the batch size and timeout.
//
// The batch collector is run in a goroutine which must be terminated with [Batcher.Shutdown].
func NewBatcher[Q, R any, C comparable, QQ ~[]Q, RR ~[]R](
	processJobs func(jobs QQ) (RR, error),
	correlateRequest func(request Q) C,
	correlateResult func(result R) C,
	opts ...Option,
) *Batcher[Q, R] {
	// Wrap the supplied processor.
	p := processor.Processor[Q, R, C, QQ, RR]{
		ProcessJobs:    processJobs,
		CorrelateQ:     correlateRequest,
		CorrelateR:     correlateResult,
		ErrNoResult:    ErrNoResult,
		ErrDuplicateID: ErrDuplicateID,
	}

	var option options
	for _, opt := range opts {
		opt.apply(&option)
	}

	queue := make(chan []internal.BatchRequest[Q, R], 1)
	queue <- nil

	return &Batcher[Q, R]{
		process: p.Process,
		queue:   queue,

		batchSize:     option.size,
		batchDuration: option.timeout,

		newTimer: timer.New,
	}
}

// options defines configurable parameters for the batcher.
type options struct {
	size    int
	timeout time.Duration
}

// Option defines configurations for [NewBatcher].
type Option interface {
	apply(opts *options)
}

// WithSize is an option to configure the batch size.
func WithSize(size int) Option {
	return sizeOption{size: size}
}

type sizeOption struct {
	size int
}

func (o sizeOption) apply(opts *options) {
	opts.size = o.size
}

// WithTimeout is an option to configure the batch timeout.
func WithTimeout(timeout time.Duration) Option {
	return timeoutOption{timeout: timeout}
}

type timeoutOption struct {
	timeout time.Duration
}

func (o timeoutOption) apply(opts *options) {
	opts.timeout = o.timeout
}

// Submit submits a job without waiting for the result.
func (b *Batcher[Q, R]) Submit(request Q) promise.Future[R] {
	result, future := promise.New[R]()
	batchRequest := internal.BatchRequest[Q, R]{
		Request: request,
		Result:  result,
	}

	b.enqueue(batchRequest)

	return future
}

// Execute submits a job and waits for the result.
func (b *Batcher[Q, R]) Execute(ctx context.Context, request Q) (R, error) {
	return b.Submit(request).Await(ctx)
}

// Send sends a batch early.
func (b *Batcher[_, _]) Send() {
	batch := <-b.queue
	b.stopTimer()
	b.queue <- nil

	if len(batch) > 0 {
		go b.process(batch)
	}
}

func (b *Batcher[Q, R]) enqueue(batchRequest internal.BatchRequest[Q, R]) {
	batch := <-b.queue
	batch = append(batch, batchRequest)

	switch len(batch) {
	case b.batchSize:
		b.stopTimer()
		go b.process(batch)
		batch = nil

	case 1:
		b.startTimer()
	}

	b.queue <- batch
}

func (b *Batcher[_, _]) startTimer() {
	if b.batchDuration <= 0 {
		return
	}

	b.timer = b.newTimer(b.batchDuration, b.timedSend)
}

func (b *Batcher[_, _]) timedSend(sent *bool) {
	batch := <-b.queue
	if *sent {
		b.queue <- batch

		return
	}

	b.queue <- nil
	if len(batch) > 0 {
		go b.process(batch)
	}
}

func (b *Batcher[_, _]) stopTimer() {
	if b.timer == nil {
		return
	}

	b.timer.Stop()
	b.timer = nil
}
