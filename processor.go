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
	"log/slog"
)

func process[Q, S Correlatable[K], K comparable](
	processor BatchProcessor[Q, S],
	request []batchRequest[Q, S],
) {
	jobs, c := separateJobs(request)

	results, err := processor.ProcessJobs(jobs)
	if err != nil {
		sendError[S, K](c, err)

		return
	}

	sendResults[S, K](c, results)
	sendError[S, K](c, ErrNoResult)
}

func separateJobs[Q Correlatable[K], S any, K comparable](
	request []batchRequest[Q, S],
) ([]Q, map[K]chan<- batchResult[S]) {
	jobs := make([]Q, 0, len(request))
	c := make(map[K]chan<- batchResult[S], len(request))
	for _, job := range request {
		jobRequest := job.request
		jobs = append(jobs, jobRequest)
		id := jobRequest.CorrelationID()
		c[id] = job.resultChan
	}

	return jobs, c
}

func sendResults[S Correlatable[K], K comparable](c map[K]chan<- batchResult[S], results []S) {
	for _, result := range results {
		id := result.CorrelationID()
		resultChan, ok := c[id]
		if ok {
			result := batchResult[S]{
				result: result,
				err:    nil,
			}
			resultChan <- result
			delete(c, id)
		} else {
			slog.Warn("Uncorrelated result dropped", "id", id)
		}
	}
}

func sendError[S any, K comparable](c map[K]chan<- batchResult[S], err error) {
	for _, ch := range c {
		ch <- batchResult[S]{
			result: *new(S),
			err:    err,
		}
	}
}
