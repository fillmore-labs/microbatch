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
	"sync"
	"testing"
	"time"

	"fillmore-labs.com/microbatch"
	"fillmore-labs.com/microbatch/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type BatcherTestSuite struct {
	suite.Suite
	BatchProcessor *mocks.MockBatchProcessor[[]int, []string]
	Batcher        *microbatch.Batcher[int, string]
}

func TestBatcherTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BatcherTestSuite))
}

func (s *BatcherTestSuite) SetupTest() {
	s.BatchProcessor = mocks.NewMockBatchProcessor[[]int, []string](s.T())

	s.Batcher = microbatch.NewBatcher(
		s.BatchProcessor,
		correlateRequest,
		correlateResult,
	)
}

func correlateRequest(q int) string {
	return strconv.Itoa(q)
}

func correlateResult(s string) string {
	return s
}

const settleGoRoutines = 10 * time.Millisecond

func (s *BatcherTestSuite) TestBatcher() {
	// given
	const iterations = 5
	returned := makeResults(iterations)
	rand.Shuffle(iterations, reflect.Swapper(returned))

	s.BatchProcessor.EXPECT().ProcessJobs(mock.Anything).Return(returned, nil).Once()

	// when
	ctx := context.Background()
	var wg sync.WaitGroup

	results := make([]string, iterations)
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if res, err := s.Batcher.ExecuteJob(ctx, i+1); err == nil {
				results[i] = res
			}
		}(i)
	}

	time.Sleep(settleGoRoutines)
	s.Batcher.Shutdown()

	// then
	wg.Wait()

	expected := makeResults(iterations)
	s.Equal(expected, results)
}

func makeResults(iterations int) []string {
	res := make([]string, 0, iterations)
	for i := 0; i < iterations; i++ {
		res = append(res, strconv.Itoa(i+1))
	}

	return res
}

func (s *BatcherTestSuite) TestTerminatedBatcher() {
	// given
	s.Batcher.Shutdown()
	s.Batcher.Shutdown()

	// when
	ctx := context.Background()
	_, err := s.Batcher.ExecuteJob(ctx, 1)

	// then
	s.ErrorIs(err, microbatch.ErrBatcherTerminated)
	s.BatchProcessor.AssertNotCalled(s.T(), "ProcessJobs", mock.Anything)
}

func (s *BatcherTestSuite) TestCancellation() {
	// given
	const iterations = 5

	s.BatchProcessor.EXPECT().ProcessJobs(mock.Anything).Return([]string{}, nil).Maybe()

	// when
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	results := make([]error, iterations)
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.Batcher.ExecuteJob(ctx, i+1)
			results[i] = err
		}(i)
	}

	cancel()
	time.Sleep(settleGoRoutines)

	s.Batcher.Shutdown()

	// then
	wg.Wait()

	for _, err := range results {
		s.ErrorIs(err, context.Canceled)
	}
}
