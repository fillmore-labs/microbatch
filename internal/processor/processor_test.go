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
	"testing"

	"fillmore-labs.com/microbatch/internal/mocks"
	"fillmore-labs.com/microbatch/internal/processor"
	internal "fillmore-labs.com/microbatch/internal/types"
	"fillmore-labs.com/promise"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var (
	errNoResult    = errors.New("no result")
	errDuplicateID = errors.New("duplicate ID")
	errTest        = errors.New("test error")
)

func correlateRequest(q int) string {
	return strconv.Itoa(q)
}

func correlateResult(r string) string {
	return r
}

type ProcessorTestSuite struct {
	suite.Suite
	BatchProcessor *mocks.MockBatchProcessor[[]int, []string]
	Processor      *processor.Processor[int, string, string, []int, []string]
}

func TestProcessorTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ProcessorTestSuite))
}

func (s *ProcessorTestSuite) SetupTest() {
	batchProcessor := mocks.NewMockBatchProcessor[[]int, []string](s.T())

	s.BatchProcessor = batchProcessor
	s.Processor = &processor.Processor[int, string, string, []int, []string]{
		ProcessJobs:    batchProcessor.ProcessJobs,
		CorrelateQ:     correlateRequest,
		CorrelateR:     correlateResult,
		ErrNoResult:    errNoResult,
		ErrDuplicateID: errDuplicateID,
	}
}

func (s *ProcessorTestSuite) TestProcessor() {
	// given
	ids := []int{1, 2, 3}
	reply := []string{"2", "1", "3"}

	s.BatchProcessor.EXPECT().ProcessJobs(mock.Anything).Return(reply, nil).Once()
	requests, results := makeRequestsResults[int, string](ids)

	// when
	s.Processor.Process(requests)

	// then
	for i, result := range results {
		value, err := result.Try()
		if s.NoErrorf(err, "failed to receive result for job %d", i+1) {
			s.Equalf(correlateRequest(i+1), value, "Unexpected result for job %d", i+1)
		}
	}
}

func (s *ProcessorTestSuite) TestProcessorError() {
	// given
	ids := []int{1, 2, 3}
	reply := errTest

	s.BatchProcessor.EXPECT().ProcessJobs(mock.Anything).Return(nil, reply).Once()
	requests, results := makeRequestsResults[int, string](ids)

	// when
	s.Processor.Process(requests)

	// then
	for i, result := range results {
		_, err := result.Try()
		s.ErrorIsf(err, errTest, "failed to receive error for job %d", i+1)
	}
}

func (s *ProcessorTestSuite) TestProcessorDuplicateUncorrelated() {
	// given
	ids := []int{1, 2, 3, 2}
	reply := []string{"3", "4", "1"}

	requests, results := makeRequestsResults[int, string](ids)
	s.BatchProcessor.EXPECT().ProcessJobs(mock.Anything).Return(reply, nil).Once()

	// when
	s.Processor.Process(requests)

	// then
	for i, result := range results {
		value, err := result.Try()
		switch i {
		case 1:
			s.ErrorIsf(err, errNoResult, "failed to receive error for job %d", i+1)
		case 3:
			s.ErrorIsf(err, errDuplicateID, "failed to receive error for job %d", i+1)
		default:
			if s.NoErrorf(err, "failed to receive result for job %d", i+1) {
				s.Equalf(correlateRequest(i+1), value, "Unexpected result for job %d", i+1)
			}
		}
	}
}

func makeRequestsResults[Q, R any](ids []Q) ([]internal.BatchRequest[Q, R], []promise.Future[R]) {
	requests := make([]internal.BatchRequest[Q, R], len(ids))
	results := make([]promise.Future[R], len(ids))
	for i, id := range ids {
		p, f := promise.New[R]()
		requests[i] = internal.BatchRequest[Q, R]{Request: id, Result: p}
		results[i] = f
	}

	return requests, results
}
