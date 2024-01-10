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

package processor_test

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"

	"fillmore-labs.com/microbatch/internal/mocks"
	"fillmore-labs.com/microbatch/internal/processor"
	internal "fillmore-labs.com/microbatch/internal/types"
	"fillmore-labs.com/microbatch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	errNoResult = errors.New("no result")
	errTest     = errors.New("test error")
)

func TestProcessor(t *testing.T) {
	t.Parallel()

	// given
	batchProcessor := mocks.NewMockBatchProcessor[[]int, []string](t)
	batchProcessor.EXPECT().ProcessJobs(mock.Anything).Return([]string{"2", "1", "3"}, nil).Once()

	p := &processor.Processor[int, string, string, []int, []string]{
		Processor:   batchProcessor,
		CorrelateQ:  strconv.Itoa,
		CorrelateS:  strings.Clone,
		ErrNoResult: errNoResult,
	}

	requests := make([]internal.BatchRequest[int, string], 0, 3)
	results := make([]<-chan types.BatchResult[string], 0, 3)
	for i := 0; i < 3; i++ {
		result := make(chan types.BatchResult[string], 1)
		results = append(results, result)
		requests = append(requests, internal.BatchRequest[int, string]{Request: i + 1, ResultChan: result})
	}

	var wg sync.WaitGroup

	// when
	wg.Add(1)
	go p.Process(requests, &wg)

	// then
	for i, ch := range results {
		value, err := (<-ch).Result()
		if assert.NoErrorf(t, err, "failed to receive result for job %d", i+1) {
			assert.Equalf(t, strconv.Itoa(i+1), value, "Unexpected result for job %d", i+1)
		}
	}
	wg.Wait()
}

func TestProcessorError(t *testing.T) {
	t.Parallel()

	// given
	batchProcessor := mocks.NewMockBatchProcessor[[]int, []string](t)
	batchProcessor.EXPECT().ProcessJobs(mock.Anything).Return(nil, errTest).Once()

	p := &processor.Processor[int, string, string, []int, []string]{
		Processor:   batchProcessor,
		CorrelateQ:  strconv.Itoa,
		CorrelateS:  strings.Clone,
		ErrNoResult: errNoResult,
	}

	requests := make([]internal.BatchRequest[int, string], 0, 3)
	results := make([]<-chan types.BatchResult[string], 0, 3)
	for i := 0; i < 3; i++ {
		result := make(chan types.BatchResult[string], 1)
		results = append(results, result)
		requests = append(requests, internal.BatchRequest[int, string]{Request: i + 1, ResultChan: result})
	}

	var wg sync.WaitGroup

	// when
	wg.Add(1)
	go p.Process(requests, &wg)

	// then
	for i, ch := range results {
		_, err := (<-ch).Result()
		assert.ErrorIsf(t, err, errTest, "failed to receive error for job %d", i+1)
	}
	wg.Wait()
}

func TestProcessorUncorrelated(t *testing.T) {
	t.Parallel()

	// given
	batchProcessor := mocks.NewMockBatchProcessor[[]int, []string](t)
	batchProcessor.EXPECT().ProcessJobs(mock.Anything).Return([]string{"3", "4", "1"}, nil).Once()

	p := &processor.Processor[int, string, string, []int, []string]{
		Processor:   batchProcessor,
		CorrelateQ:  strconv.Itoa,
		CorrelateS:  strings.Clone,
		ErrNoResult: errNoResult,
	}

	requests := make([]internal.BatchRequest[int, string], 0, 3)
	results := make([]<-chan types.BatchResult[string], 0, 3)
	for i := 0; i < 3; i++ {
		result := make(chan types.BatchResult[string], 1)
		results = append(results, result)
		requests = append(requests, internal.BatchRequest[int, string]{Request: i + 1, ResultChan: result})
	}

	var wg sync.WaitGroup

	// when
	wg.Add(1)
	go p.Process(requests, &wg)

	// then
	for i, ch := range results {
		value, err := (<-ch).Result()
		switch i {
		case 1:
			assert.ErrorIsf(t, err, errNoResult, "failed to receive error for job %d", i+1)
		default:
			if assert.NoErrorf(t, err, "failed to receive result for job %d", i+1) {
				assert.Equalf(t, strconv.Itoa(i+1), value, "Unexpected result for job %d", i+1)
			}
		}
	}
	wg.Wait()
}
