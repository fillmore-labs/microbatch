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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fillmore-labs.com/microbatch"
	"fillmore-labs.com/microbatch/dataloader"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
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

	g, ctx := errgroup.WithContext(context.Background())
	for _, query := range []int{1, 2, 1, 2, 3, 3, 4, 1, 2, 3, 5} {
		query := query
		future := d.Load(query)
		g.Go(func() error {
			_, err := future.Wait(ctx)

			return err
		})
	}

	d.Shutdown()

	// Wait for all queries to complete
	if err := g.Wait(); err != nil {
		panic(err)
	}

	fmt.Printf("Requested %d keys in %d calls\n", p.Keys.Load(), p.Calls.Load())
	// Output:
	// Requested 5 keys in 2 calls
}

const settleGoRoutines = 10 * time.Millisecond

func TestCancellation(t *testing.T) {
	t.Parallel()

	// given
	p := &DataProcessor{}
	d := dataloader.NewDataLoader(
		p.ProcessJobs,
		func(q QueryResult) int { return q.ID },
		// unbounded, won't process before shutdown
	)

	// when
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	queries := [11]int{1, 2, 1, 2, 3, 3, 4, 1, 2, 3, 5}
	var errors [len(queries)]error
	for i, query := range queries {
		future := d.Load(query)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errors[i] = future.Wait(ctx)
		}(i)
	}

	cancel() // cancel all waits

	time.Sleep(settleGoRoutines)
	d.Shutdown()

	// then
	wg.Wait()

	for i, err := range errors {
		assert.ErrorIsf(t, err, context.Canceled, "Unexpected result for job %d", i+1)
	}
}
