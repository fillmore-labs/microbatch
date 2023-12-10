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
	"log/slog"
)

type processor[Q, S any, K comparable, QQ ~[]Q, SS ~[]S] struct {
	processor  BatchProcessor[QQ, SS]
	correlateQ func(job Q) K
	correlateS func(jobResult S) K
}

func (p *processor[Q, S, K, QQ, SS]) process(request []bRequest[Q, S]) {
	jobs, resultChannels := separateJobs(request, p.correlateQ)

	results, err := p.processor.ProcessJobs(jobs)
	if err != nil {
		sendError(err, resultChannels)

		return
	}

	sendResults(results, resultChannels, p.correlateS)
	sendError(ErrNoResult, resultChannels)
}

func separateJobs[Q, S any, K comparable](
	request []bRequest[Q, S],
	correlateQ func(Q) K,
) ([]Q, map[K]chan<- batchResult[S]) {
	jobs := make([]Q, 0, len(request))
	resultChannels := make(map[K]chan<- batchResult[S], len(request))

	for _, job := range request {
		jobRequest := job.request
		jobs = append(jobs, jobRequest)

		correlationID := correlateQ(jobRequest)
		resultChannels[correlationID] = job.resultChan
	}

	return jobs, resultChannels
}

func sendResults[S any, K comparable](
	results []S,
	resultChannels map[K]chan<- batchResult[S],
	correlateS func(S) K,
) {
	for _, result := range results {
		correlationID := correlateS(result)
		resultChan, ok := resultChannels[correlationID]
		if ok {
			resultChan <- bResult[S]{
				result: result,
				err:    nil,
			}
			delete(resultChannels, correlationID)
		} else {
			slog.Warn("Uncorrelated result dropped", "id", correlationID)
		}
	}
}

func sendError[S any, K comparable](err error, resultChannels map[K]chan<- batchResult[S]) {
	for _, resultChan := range resultChannels {
		resultChan <- bResult[S]{
			result: *new(S),
			err:    err,
		}
	}
}
