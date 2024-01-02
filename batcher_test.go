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

package microbatch_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fillmore-labs.com/microbatch"
)

func Example_execute() {
	// Initialize
	processor := &RemoteProcessor{}
	opts := []microbatch.Option{microbatch.WithSize(3), microbatch.WithTimeout(10 * time.Millisecond)}
	batcher := microbatch.NewBatcher(
		processor,
		func(j *Job) JobID { return j.ID },
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

func Example_submit() {
	// Initialize
	processor := &RemoteProcessor{}
	batcher := microbatch.NewBatcher(
		processor,
		func(j *Job) JobID { return j.ID },
		func(r *JobResult) JobID { return r.ID },
		microbatch.WithSize(3),
	)

	ctx := context.Background()
	ctx2, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
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
		go func(i int, result <-chan microbatch.BatchResult[*JobResult]) {
			defer wg.Done()

			select {
			case r := <-result:
				value, err := r.Result()
				if err == nil {
					fmt.Println(value.Body)
				} else {
					fmt.Printf("Error executing job %d: %v\n", i, err)
				}

			case <-ctx2.Done():
				fmt.Printf("Job %d canceled: %v\n", i, ctx2.Err())
			}
		}(i, result)
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
