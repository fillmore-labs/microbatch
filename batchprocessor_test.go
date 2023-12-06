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
	"fmt"
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
