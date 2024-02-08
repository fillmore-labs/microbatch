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
	"errors"
	"time"

	"fillmore-labs.com/exp/async"
	"fillmore-labs.com/microbatch/internal/collector"
	"fillmore-labs.com/microbatch/internal/processor"
	internal "fillmore-labs.com/microbatch/internal/types"
)

// Batcher handles submitting requests in batches and returning results through channels.
type Batcher[Q, R any] struct {
	requests chan<- internal.BatchRequest[Q, R]

	terminating chan<- struct{}
	terminated  <-chan struct{}
}

var (
	// ErrBatcherTerminated is returned when the batcher is terminated.
	ErrBatcherTerminated = errors.New("batcher terminated")
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
	// Channels used for communicating from the Batcher to the Collector.
	requests := make(chan internal.BatchRequest[Q, R])
	terminating := make(chan struct{})
	terminated := make(chan struct{})

	// Wrap the supplied processor.
	p := &processor.Processor[Q, R, C, QQ, RR]{
		ProcessJobs:    processJobs,
		CorrelateQ:     correlateRequest,
		CorrelateR:     correlateResult,
		ErrNoResult:    ErrNoResult,
		ErrDuplicateID: ErrDuplicateID,
	}

	option := options{}
	for _, opt := range opts {
		opt(&option)
	}

	c := &collector.Collector[Q, R]{
		Requests:      requests,
		Terminating:   terminating,
		Terminated:    terminated,
		Processor:     p,
		BatchSize:     option.size,
		BatchDuration: option.timeout,
	}

	go c.Run()

	return &Batcher[Q, R]{
		requests:    requests,
		terminating: terminating,
		terminated:  terminated,
	}
}

// options defines configurable parameters for the batcher.
type options struct {
	size    int
	timeout time.Duration
}

// Option defines configurations for [NewBatcher].
type Option func(*options)

// WithSize is an option to configure the batch size.
func WithSize(size int) Option {
	return func(o *options) {
		o.size = size
	}
}

// WithTimeout is an option to configure the batch timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

// SubmitJob Submits a job without waiting for the result.
func (b *Batcher[Q, R]) SubmitJob(request Q) async.Future[R] {
	future, promise := async.NewFuture[R]()
	batchRequest := internal.BatchRequest[Q, R]{
		Request: request,
		Result:  promise,
	}

	select {
	case b.requests <- batchRequest:

	case <-b.terminated:
		promise.Reject(ErrBatcherTerminated)
	}

	return future
}

// Shutdown needs to be called to reclaim resources and send the last batch.
// No calls to [Batcher.SubmitJob] after this will be accepted.
func (b *Batcher[_, _]) Shutdown() {
	select {
	case b.terminating <- struct{}{}:
		<-b.terminated

	case <-b.terminated:
	}
}
