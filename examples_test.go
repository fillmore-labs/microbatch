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

package microbatch_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fillmore-labs.com/exp/async"
	"fillmore-labs.com/microbatch"
)

type (
	JobID int

	Job struct {
		ID JobID
	}

	JobResult struct {
		ID   JobID
		Body string
	}

	Jobs       []*Job
	JobResults []*JobResult
)

func (j *Job) JobID() JobID                  { return j.ID }
func (j *JobResult) JobID() JobID            { return j.ID }
func (j *JobResult) Unwrap() (string, error) { return j.Body, nil }

type RemoteProcessor struct{}

func (p *RemoteProcessor) ProcessJobs(jobs Jobs) (JobResults, error) {
	results := make(JobResults, 0, len(jobs))
	for _, job := range jobs {
		result := &JobResult{
			ID:   job.ID,
			Body: fmt.Sprintf("Processed job %d", job.ID),
		}
		results = append(results, result)
	}

	return results, nil
}

// Example (Blocking) demonstrates how to use [Batcher.SubmitJob] in a single line.
func Example_blocking() {
	// Initialize
	processor := &RemoteProcessor{}
	batcher := microbatch.NewBatcher(
		processor.ProcessJobs,
		(*Job).JobID,
		(*JobResult).JobID,
		microbatch.WithSize(3),
		microbatch.WithTimeout(10*time.Millisecond),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	const iterations = 5
	var wg sync.WaitGroup

	// Submit jobs
	for i := 1; i <= iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if result, err := batcher.SubmitJob(&Job{ID: JobID(i)}).Wait(ctx); err == nil {
				fmt.Println(result.Body)
			}
		}(i) // https://go.dev/doc/faq#closures_and_goroutines
	}

	// Shut down
	wg.Wait()
	batcher.Shutdown()
	// Unordered output:
	// Processed job 1
	// Processed job 2
	// Processed job 3
	// Processed job 4
	// Processed job 5
}

// Example (Asynchronous) demonstrates how to use [Batcher.SubmitJob] with a timeout.
// Note that you can shut down the batcher without waiting for the jobs to finish.
func Example_asynchronous() {
	// Initialize
	processor := &RemoteProcessor{}
	batcher := microbatch.NewBatcher(
		processor.ProcessJobs,
		(*Job).JobID,
		(*JobResult).JobID,
		microbatch.WithSize(3),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const iterations = 5

	var wg sync.WaitGroup
	for i := 1; i <= iterations; i++ {
		future := batcher.SubmitJob(&Job{ID: JobID(i)})

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			result, err := async.Then(ctx, future, (*JobResult).Unwrap)
			if err == nil {
				fmt.Println(result)
			} else {
				fmt.Printf("Error executing job %d: %v\n", i, err)
			}
		}(i)
	}

	// Shut down
	batcher.Shutdown()
	wg.Wait()
	// Unordered output:
	// Processed job 1
	// Processed job 2
	// Processed job 3
	// Processed job 4
	// Processed job 5
}
