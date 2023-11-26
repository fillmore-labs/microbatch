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

package microbatch_test

import (
	"fmt"

	"fillmore-labs.com/microbatch"
)

type RemoteProcessor struct{}

func (p *RemoteProcessor) ProcessJobs(jobs []*Job) ([]*JobResult, error) {
	result := make([]*JobResult, 0, len(jobs))
	for _, job := range jobs {
		body := fmt.Sprintf("Processed job %d", job.ID)
		result = append(result, &JobResult{ID: job.ID, Body: body})
	}

	return result, nil
}

func ExampleBatchProcessor() {
	var processor microbatch.BatchProcessor[*Job, *JobResult] = &RemoteProcessor{}

	result, _ := processor.ProcessJobs([]*Job{})
	fmt.Println(result)
	// Output:
	// []
}
