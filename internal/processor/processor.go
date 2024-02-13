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
	"fillmore-labs.com/promise"
)

// Processor handles batch processing of jobs and results.
//
// This structure serves two purposes:
//   - It is read-only after construction and therefore thread-safe.
//   - It isolates collector logic from correlation types.
type Processor[Q, R any, C comparable, QQ ~[]Q, RR ~[]R] struct {
	// ProcessJobs processes job batches.
	ProcessJobs func(jobs QQ) (RR, error)
	// CorrelateQ maps each job to a correlation ID.
	CorrelateQ func(job Q) C
	// CorrelateR maps each result to a correlation ID.
	CorrelateR func(jobResult R) C
	// ErrNoResult is sent if there is no matching result for a job.
	ErrNoResult error
	// ErrDuplicateID is sent if there is a duplicate correlation ID.
	ErrDuplicateID error
}

// resultMap is map from correlation ID to result [promise.Promise].
type resultMap[R any, C comparable] map[C]promise.Promise[R]

// Process takes a batch of requests and handles processing.
func (p *Processor[Q, R, _, _, _]) Process(requests []internal.BatchRequest[Q, R]) {
	// Separate jobs from result channels.
	jobs, promises := p.separateJobs(requests)

	// Process jobs.
	results, err := p.ProcessJobs(jobs)
	if err != nil { // Send errors if processing failed.
		promises.sendError(err)

		return
	}

	// Send successful results.
	promises.sendResults(results, p.CorrelateR)
	// Send errors for jobs without results.
	promises.sendError(p.ErrNoResult)
}

// separateJobs separates jobs from result channels.
func (p *Processor[Q, R, C, QQ, _]) separateJobs(
	requests []internal.BatchRequest[Q, R],
) (QQ, resultMap[R, C]) {
	jobs := make(QQ, 0, len(requests))
	promises := make(resultMap[R, C], len(requests))

	for _, job := range requests {
		jobRequest, jobResult := job.Request, job.Result

		correlationID := p.CorrelateQ(jobRequest)
		if _, ok := promises[correlationID]; ok {
			jobResult.Reject(fmt.Errorf("%w: %v", p.ErrDuplicateID, correlationID))

			continue
		}

		jobs = append(jobs, jobRequest)
		promises[correlationID] = jobResult
	}

	return jobs, promises
}

// sendResults sends results to matching channels.
func (r resultMap[R, C]) sendResults(
	results []R,
	correlateR func(jobResult R) C,
) {
	for _, result := range results {
		correlationID := correlateR(result)
		promise, ok := r[correlationID]
		if !ok {
			slog.Warn("Uncorrelated result dropped", "id", correlationID)

			continue
		}

		delete(r, correlationID)
		promise.Resolve(result)
	}
}

// sendError sends an error to all remaining result channels.
func (r resultMap[R, _]) sendError(err error) {
	for _, promise := range r {
		promise.Reject(err)
	}
}
