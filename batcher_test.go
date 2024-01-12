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
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"fillmore-labs.com/microbatch"
	"fillmore-labs.com/microbatch/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const settleGoRoutines = 10 * time.Millisecond

func makeResults(iterations int) []string {
	res := make([]string, 0, iterations)
	for i := 0; i < iterations; i++ {
		res = append(res, strconv.Itoa(i+1))
	}

	return res
}

func TestBatcher(t *testing.T) {
	t.Parallel()

	// given
	const iterations = 5
	returned := makeResults(iterations)
	rand.Shuffle(iterations, reflect.Swapper(returned))

	batchProcessor := mocks.NewMockBatchProcessor[[]int, []string](t)
	batchProcessor.EXPECT().ProcessJobs(mock.Anything).Return(returned, nil).Once()

	batcher := microbatch.NewBatcher(
		batchProcessor,
		strconv.Itoa,
		strings.Clone,
	)

	// when
	ctx := context.Background()
	var wg sync.WaitGroup

	results := make([]string, iterations)
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if res, err := batcher.ExecuteJob(ctx, i+1); err == nil {
				results[i] = res
			}
		}(i)
	}

	time.Sleep(settleGoRoutines)
	batcher.Shutdown()

	// then
	wg.Wait()

	expected := makeResults(iterations)
	assert.Equal(t, expected, results)
}

func TestTerminatedBatcher(t *testing.T) {
	t.Parallel()

	// given
	batchProcessor := mocks.NewMockBatchProcessor[[]int, []string](t)

	batcher := microbatch.NewBatcher(
		batchProcessor,
		strconv.Itoa,
		strings.Clone,
	)

	// when
	ctx := context.Background()

	batcher.Shutdown()
	batcher.Shutdown()

	_, err := batcher.ExecuteJob(ctx, 1)

	// then
	assert.ErrorIs(t, err, microbatch.ErrBatcherTerminated)
	batchProcessor.AssertNotCalled(t, "ProcessJobs", mock.Anything)
}
