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

func Example() {
	// Initialize
	processor := &RemoteProcessor{}
	batcher := microbatch.NewBatcher(
		processor,
		func(j *Job) JobID { return j.ID },
		func(r *JobResult) JobID { return r.ID },
		3,
		10*time.Millisecond,
	)

	ctx := context.Background()
	const iterations = 5
	var wg sync.WaitGroup

	// Submit jobs
	for i := 1; i <= iterations; i++ {
		wg.Add(1)
		go func(i int) {
			result, _ := batcher.ExecuteJob(ctx, &Job{ID: JobID(i)})
			fmt.Println(result.Body)
			wg.Done()
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
