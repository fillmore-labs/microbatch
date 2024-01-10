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
	"log/slog"
	"sync"

	internal "fillmore-labs.com/microbatch/internal/types"
	"fillmore-labs.com/microbatch/types"
)

type Processor[Q, S any, K comparable, QQ ~[]Q, SS ~[]S] struct {
	Processor   types.BatchProcessor[QQ, SS]
	CorrelateQ  func(job Q) K
	CorrelateS  func(jobResult S) K
	ErrNoResult error
}

func (p *Processor[Q, S, K, QQ, SS]) Process(request []internal.BatchRequest[Q, S], wg *sync.WaitGroup) {
	defer wg.Done()
	jobs, resultChannels := separateJobs(request, p.CorrelateQ)

	results, err := p.Processor.ProcessJobs(jobs)
	if err != nil {
		sendError(err, resultChannels)

		return
	}

	sendResults(results, resultChannels, p.CorrelateS)
	sendError(p.ErrNoResult, resultChannels)
}

func separateJobs[Q, S any, K comparable](
	request []internal.BatchRequest[Q, S],
	correlateQ func(Q) K,
) ([]Q, map[K]chan<- types.BatchResult[S]) {
	jobs := make([]Q, 0, len(request))
	resultChannels := make(map[K]chan<- types.BatchResult[S], len(request))

	for _, job := range request {
		jobRequest := job.Request
		jobs = append(jobs, jobRequest)

		correlationID := correlateQ(jobRequest)
		resultChannels[correlationID] = job.ResultChan
	}

	return jobs, resultChannels
}

type batchResult[S any] struct {
	value S
	err   error
}

func (b batchResult[S]) Result() (S, error) {
	return b.value, b.err
}

func sendResults[S any, K comparable](
	results []S,
	resultChannels map[K]chan<- types.BatchResult[S],
	correlateS func(S) K,
) {
	for _, result := range results {
		correlationID := correlateS(result)
		resultChan, ok := resultChannels[correlationID]
		if ok {
			resultChan <- batchResult[S]{
				value: result,
				err:   nil,
			}
			delete(resultChannels, correlationID)
		} else {
			slog.Warn("Uncorrelated result dropped", "id", correlationID)
		}
	}
}

func sendError[S any, K comparable](err error, resultChannels map[K]chan<- types.BatchResult[S]) {
	for _, resultChan := range resultChannels {
		resultChan <- batchResult[S]{
			value: *new(S),
			err:   err,
		}
	}
}
