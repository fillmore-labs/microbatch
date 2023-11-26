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

type (
	JobID int

	Job struct {
		ID JobID
	}

	JobResult struct {
		ID   JobID
		Body string
	}
)

func (j *Job) CorrelationID() JobID {
	return j.ID
}

func (j *JobResult) CorrelationID() JobID {
	return j.ID
}

func ExampleCorrelatable() {
	printID := func(j microbatch.Correlatable[JobID]) {
		fmt.Println(j.CorrelationID())
	}

	printID(&Job{ID: 1})
	printID(&JobResult{ID: 1})
	// Output:
	// 1
	// 1
}
