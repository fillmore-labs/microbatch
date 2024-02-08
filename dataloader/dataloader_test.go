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

package dataloader_test

import (
	"context"
	"fmt"
	"sync/atomic"

	"fillmore-labs.com/exp/async"
	"fillmore-labs.com/microbatch"
	"fillmore-labs.com/microbatch/dataloader"
)

type DataProcessor struct {
	Calls atomic.Int32
	Keys  atomic.Int32
}

type QueryResult struct {
	ID    int
	Value string
}

func (p *DataProcessor) ProcessJobs(keys []int) ([]QueryResult, error) {
	p.Calls.Add(1)
	p.Keys.Add(int32(len(keys)))
	results := make([]QueryResult, 0, len(keys))
	for _, key := range keys {
		result := QueryResult{
			ID:    key,
			Value: fmt.Sprintf("Query Result %d", key),
		}
		results = append(results, result)
	}

	return results, nil
}

func Example() {
	p := &DataProcessor{}
	d := dataloader.NewDataLoader(
		p.ProcessJobs,
		func(q QueryResult) int { return q.ID },
		microbatch.WithSize(3),
	)

	queries := [11]int{1, 2, 1, 2, 3, 3, 4, 1, 2, 3, 5}
	var results [len(queries)]async.Awaitable[QueryResult]
	for i, query := range queries {
		results[i] = d.Load(query)
	}

	d.Shutdown()

	// Wait for all queries to complete
	ctx := context.Background()
	if _, err := async.WaitAllValues(ctx, results[:]...); err == nil {
		fmt.Printf("Requested %d keys in %d calls\n", p.Keys.Load(), p.Calls.Load())
	}

	// Output:
	// Requested 5 keys in 2 calls
}
