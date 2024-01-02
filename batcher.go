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
	"context"
	"errors"
	"fmt"
	"time"
)

// Batcher handles submitting requests in batches and returning results through channels.
type Batcher[Q, S any] struct {
	requests chan<- batchRequest[Q, S]
	done     chan struct{}
}

// batchRequest represents a single request submitted to the [Batcher], along with the channel to return the result on.
type batchRequest[Q, S any] struct {
	request    Q
	resultChan chan<- BatchResult[S]
}

// BatchResult defines the interface for returning results from batch processing.
type BatchResult[S any] interface {
	Result() (S, error)
}

// processor defines the interface for processing batches of requests.
type processor[Q, S any] interface {
	process(requests []batchRequest[Q, S])
}

// options defines configurable parameters for the batcher.
type options struct {
	size    int
	timeout time.Duration
}

// NewBatcher creates a new [Batcher].
func NewBatcher[Q, S any, K comparable, QQ ~[]Q, SS ~[]S](
	processor BatchProcessor[QQ, SS],
	correlateRequest func(Q) K,
	correlateResult func(S) K,
	opts ...Option,
) *Batcher[Q, S] {
	option := &options{}
	for _, opt := range opts {
		opt(option)
	}

	requests := make(chan batchRequest[Q, S])

	b := batchRunner[Q, S]{
		requests:      requests,
		batchSize:     option.size,
		batchDuration: option.timeout,
		processor: &batchProcessor[Q, S, K, QQ, SS]{
			processor:  processor,
			correlateQ: correlateRequest,
			correlateS: correlateResult,
		},
	}

	go b.runBatcher()

	return &Batcher[Q, S]{
		requests: requests,
		done:     make(chan struct{}),
	}
}

// Option defines configurable parameters for [NewBatcher].
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

// Errors returned from [Batcher.ExecuteJob].
var (
	// ErrBatcherTerminated is returned when the batcher is terminated.
	ErrBatcherTerminated = errors.New("batcher terminated")
	// ErrNoResult is returned when the response from [BatchProcessor] is missing a
	// matching correlation ID.
	ErrNoResult = errors.New("no result")
)

// ExecuteJob submits a job and waits for the result.
func (b *Batcher[Q, S]) ExecuteJob(ctx context.Context, request Q) (S, error) {
	resultChan, err := b.SubmitJob(ctx, request)
	if err != nil {
		return *new(S), err
	}

	select {
	case result := <-resultChan:
		return result.Result()

	case <-ctx.Done():
		return *new(S), fmt.Errorf("job canceled: %w", ctx.Err())
	}
}

// SubmitJob Submits a job without waiting for the result.
func (b *Batcher[Q, S]) SubmitJob(ctx context.Context, request Q) (<-chan BatchResult[S], error) {
	resultChan := make(chan BatchResult[S])

	r := batchRequest[Q, S]{
		request:    request,
		resultChan: resultChan,
	}

	select {
	case _, ok := <-b.done:
		if !ok {
			return nil, ErrBatcherTerminated
		}

	case b.requests <- r:

	case <-ctx.Done():
		return nil, fmt.Errorf("job canceled: %w", ctx.Err())
	}

	return resultChan, nil
}

// Shutdown needs to be called to reclaim resources and send the last batch.
// No calls to [Batcher.SubmitJob] or [Batcher.ExecuteJob] after this will be accepted.
func (b *Batcher[Q, S]) Shutdown() {
	close(b.done)
	close(b.requests)
	b.requests = nil
}
