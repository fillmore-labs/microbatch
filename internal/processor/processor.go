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

package processor

import (
	"fmt"
	"log/slog"

	internal "fillmore-labs.com/microbatch/internal/types"
	"fillmore-labs.com/microbatch/types"
)

// Processor handles batch processing of jobs and results.
//
// This structure serves two purposes:
//   - It is read-only after construction and therefore thread-safe.
//   - It isolates collector logic from correlation types.
type Processor[Q, R any, C comparable, QQ ~[]Q, RR ~[]R] struct {
	// Processor processes job batches.
	Processor types.BatchProcessor[QQ, RR]
	// CorrelateQ maps each job to a correlation ID.
	CorrelateQ func(job Q) C
	// CorrelateR maps each result to a correlation ID.
	CorrelateR func(jobResult R) C
	// ErrNoResult is sent if there is no matching result for a job.
	ErrNoResult error
	// ErrDuplicateID is sent if there is a duplicate correlation ID.
	ErrDuplicateID error
}

// resultChanMap is map from correlation IDs to result channels.
type resultChanMap[R any, C comparable] map[C]chan<- types.BatchResult[R]

// Process takes a batch of requests and handles processing.
func (p *Processor[Q, R, _, _, _]) Process(requests []internal.BatchRequest[Q, R]) {
	// Separate jobs from result channels.
	jobs, resultChannels := p.separateJobs(requests)

	// Process jobs.
	results, err := p.Processor.ProcessJobs(jobs)
	if err != nil { // Send errors if processing failed.
		p.sendError(resultChannels, err)

		return
	}

	// Send successful results.
	p.sendResults(results, resultChannels)
	// Send errors for jobs without results.
	p.sendError(resultChannels, p.ErrNoResult)
}

// separateJobs separates jobs from result channels.
func (p *Processor[Q, R, C, QQ, _]) separateJobs(
	requests []internal.BatchRequest[Q, R],
) (QQ, resultChanMap[R, C]) {
	jobs := make(QQ, 0, len(requests))
	resultChannels := make(resultChanMap[R, C], len(requests))

	for _, job := range requests {
		jobRequest, resultChan := job.Request, job.ResultChan

		correlationID := p.CorrelateQ(jobRequest)
		if _, ok := resultChannels[correlationID]; ok {
			resultChan <- batchResult[R]{
				err: fmt.Errorf("%w: %v", p.ErrDuplicateID, correlationID),
			}

			continue
		}

		jobs = append(jobs, jobRequest)
		resultChannels[correlationID] = resultChan
	}

	return jobs, resultChannels
}

// NewResultChannel creates a new result channel.
//
// This function is here because processor logic implicitly depends on a buffered channel to allow for sending results
// without blocking.
func NewResultChannel[R any]() chan types.BatchResult[R] {
	return make(chan types.BatchResult[R], 1)
}

// batchResult is a result for a request send to the result channel.
type batchResult[R any] struct {
	value R
	err   error
}

// Result implements [types.BatchResult].
func (b batchResult[R]) Result() (R, error) {
	return b.value, b.err
}

// sendResults sends results to matching channels.
func (p *Processor[_, R, C, _, RR]) sendResults(
	results RR,
	resultChannels resultChanMap[R, C],
) {
	for _, result := range results {
		correlationID := p.CorrelateR(result)
		resultChan, ok := resultChannels[correlationID]
		if !ok {
			slog.Warn("Uncorrelated result dropped", "id", correlationID)

			continue
		}

		delete(resultChannels, correlationID)
		resultChan <- batchResult[R]{value: result}
		close(resultChan)
	}
}

// sendError sends an error to all remaining result channels.
func (*Processor[_, R, C, _, _]) sendError(resultChannels resultChanMap[R, C], err error) {
	for _, resultChan := range resultChannels {
		resultChan <- batchResult[R]{err: err}
		close(resultChan)
	}
}
