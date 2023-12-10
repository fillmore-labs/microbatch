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

// BatchProcessor is the interface your batch processor needs to implement.
type BatchProcessor[QQ, SS any] interface {
	ProcessJobs(jobs QQ) (SS, error)
}

// Batcher is used to submit requests.
type Batcher[Q, S any] struct {
	requestChan chan<- bRequest[Q, S]
	done        chan struct{}
}

// Result from [Batcher.submitJob].
type batchResult[S any] interface {
	Result() (S, error)
}

type bRequest[Q, S any] struct {
	request    Q
	resultChan chan<- batchResult[S]
}

type bResult[S any] struct {
	result S
	err    error
}

func (b bResult[S]) Result() (S, error) {
	return b.result, b.err
}

// NewBatcher creates a new [Batcher].
func NewBatcher[Q, S any, K comparable, QQ ~[]Q, SS ~[]S](
	batchProcessor BatchProcessor[QQ, SS],
	correlateRequest func(Q) K,
	correlateResult func(S) K,
	size int,
	duration time.Duration,
) *Batcher[Q, S] {
	requestChan := make(chan bRequest[Q, S])

	b := batchRunner[Q, S, K, QQ, SS]{
		batchSize:     size,
		batchDuration: duration,
		requestChan:   requestChan,
		processor: &processor[Q, S, K, QQ, SS]{
			processor:  batchProcessor,
			correlateQ: correlateRequest,
			correlateS: correlateResult,
		},
	}

	go b.runBatcher()

	return &Batcher[Q, S]{
		requestChan: requestChan,
		done:        make(chan struct{}),
	}
}

// Shutdown needs to be called to send the last batch and terminate the goroutine.
// No calls to [Batcher.ExecuteJob] after this will be accepted.
func (b *Batcher[Q, S]) Shutdown() {
	close(b.done)

	rc := b.requestChan
	b.requestChan = nil
	close(rc)
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
	resultChan := make(chan batchResult[S])

	err := b.submitJob(ctx, request, resultChan)
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

// Submit a job.
func (b *Batcher[Q, S]) submitJob(ctx context.Context, r Q, resultChan chan<- batchResult[S]) error {
	select {
	case _, ok := <-b.done:
		if !ok {
			return ErrBatcherTerminated
		}

	case b.requestChan <- bRequest[Q, S]{
		request:    r,
		resultChan: resultChan,
	}:

	case <-ctx.Done():
		return fmt.Errorf("job canceled: %w", ctx.Err())
	}

	return nil
}
