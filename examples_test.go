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

	"fillmore-labs.com/microbatch"
	"fillmore-labs.com/microbatch/types"
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

var _ types.BatchProcessor[Jobs, JobResults] = (*RemoteProcessor)(nil)

func ExampleBatchProcessor() {
	processor := &RemoteProcessor{}
	results, _ := processor.ProcessJobs(Jobs{&Job{ID: 1}, &Job{ID: 2}})
	for _, result := range results {
		fmt.Println(result.Body)
	}
	// Output:
	// Processed job 1
	// Processed job 2
}

// Example (ExecuteJob) demonstrates how to use the blocking [Batcher.ExecuteJob].
func Example_executeJob() {
	// Initialize
	processor := &RemoteProcessor{}
	opts := []microbatch.Option{microbatch.WithSize(3), microbatch.WithTimeout(10 * time.Millisecond)}
	batcher := microbatch.NewBatcher(
		processor,
		func(q *Job) JobID { return q.ID },
		func(r *JobResult) JobID { return r.ID },
		opts...,
	)

	ctx := context.Background()
	const iterations = 5
	var wg sync.WaitGroup

	// Submit jobs
	for i := 1; i <= iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if result, err := batcher.ExecuteJob(ctx, &Job{ID: JobID(i)}); err == nil {
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

// Example (SubmitJob) demonstrates how to use [Batcher.SubmitJob] with a timeout.
// Note that you can shut down the batcher without waiting for the jobs to finish.
func Example_submitJob() {
	// Initialize
	processor := &RemoteProcessor{}
	batcher := microbatch.NewBatcher(
		processor,
		func(q *Job) JobID { return q.ID },
		func(r *JobResult) JobID { return r.ID },
		microbatch.WithSize(3),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	const iterations = 5

	var wg sync.WaitGroup
	for i := 1; i <= iterations; i++ {
		result, err := batcher.SubmitJob(ctx, &Job{ID: JobID(i)})
		if err != nil {
			fmt.Printf("Error submitting job %d: %v\n", i, err)

			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			select {
			case r := <-result:
				value, err := r.Result()
				if err == nil {
					fmt.Println(value.Body)
				} else {
					fmt.Printf("Error executing job %d: %v\n", i, err)
				}

			case <-ctx.Done():
				fmt.Printf("Job %d canceled: %v\n", i, ctx.Err())
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
