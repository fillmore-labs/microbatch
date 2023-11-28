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

type processor[Q, S any, K comparable, QQ ~[]Q, SS ~[]S] struct {
	processor  BatchProcessor[Q, S, QQ, SS]
	correlateQ func(job Q) K
	correlateS func(jobResult S) K
}

func (p *processor[Q, S, K, QQ, SS]) process(request []batchRequest[Q, S]) {
	jobs, c := p.separateJobs(request)

	results, err := p.processor.ProcessJobs(jobs)
	if err != nil {
		p.sendError(c, err)

		return
	}

	p.sendResults(c, results)
	p.sendError(c, ErrNoResult)
}

func (p *processor[Q, S, K, QQ, SS]) separateJobs(
	request []batchRequest[Q, S],
) ([]Q, map[K]chan<- batchResult[S]) {
	jobs := make([]Q, 0, len(request))
	c := make(map[K]chan<- batchResult[S], len(request))

	for _, job := range request {
		jobRequest := job.request
		jobs = append(jobs, jobRequest)
		id := p.correlateQ(jobRequest)
		c[id] = job.resultChan
	}

	return jobs, c
}

func (p *processor[Q, S, K, QQ, SS]) sendResults(c map[K]chan<- batchResult[S], results []S) {
	for _, result := range results {
		id := p.correlateS(result)
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

func (*processor[Q, S, K, QQ, SS]) sendError(c map[K]chan<- batchResult[S], err error) {
	for _, ch := range c {
		ch <- batchResult[S]{
			result: *new(S),
			err:    err,
		}
	}
}
