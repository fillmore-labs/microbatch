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
	"golang.org/x/sync/errgroup"
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
	batchProcessor := mocks.NewMockBatchProcessor[[]int, []string](s.T())

	s.BatchProcessor = batchProcessor
	s.Batcher = microbatch.NewBatcher(
		batchProcessor.ProcessJobs,
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
	var g errgroup.Group

	var results [iterations]string
	for i := 0; i < iterations; i++ {
		i := i
		g.Go(func() error {
			res, err := s.Batcher.SubmitJob(i + 1).Wait(ctx)
			if err == nil {
				results[i] = res
			}

			return err
		})
	}

	time.Sleep(settleGoRoutines)
	s.Batcher.Shutdown()

	// then
	err := g.Wait()
	s.NoErrorf(err, "Unexpected error executing jobs")

	expected := makeResults(iterations)
	s.Equal(expected, results[:])
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
	_, err := s.Batcher.SubmitJob(1).Wait(ctx)

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

	var errors [iterations]error
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errors[i] = s.Batcher.SubmitJob(i + 1).Wait(ctx)
		}(i)
	}

	cancel()
	time.Sleep(settleGoRoutines)

	s.Batcher.Shutdown()

	// then
	wg.Wait()

	for i, err := range errors {
		s.ErrorIsf(err, context.Canceled, "Unexpected result for job %d", i+1)
	}
}
