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
	"testing"
	"time"

	"fillmore-labs.com/microbatch"
	"fillmore-labs.com/microbatch/internal/mocks"
	"fillmore-labs.com/microbatch/internal/timer"
	internal "fillmore-labs.com/microbatch/internal/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TimedBatcherTestSuite struct {
	suite.Suite
	Processor *mocks.MockProcessor[int, string]
	Batcher   *microbatch.Batcher[int, string]

	Timer    *mocks.MockTimer
	NewTimer *mocks.MockNewTimer
}

func TestTimedBatcherTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TimedBatcherTestSuite))
}

const (
	batchSize     = 5
	batchDuration = 1 * time.Second
)

func (s *TimedBatcherTestSuite) SetupTest() {
	mp := mocks.NewMockProcessor[int, string](s.T())
	mt := mocks.NewMockTimer(s.T())
	mnt := mocks.NewMockNewTimer(s.T())

	s.Processor = mp
	s.Timer = mt
	s.NewTimer = mnt

	s.Batcher = microbatch.NewTestBatcher(
		mp.Process,
		mnt.New,
		microbatch.WithSize(batchSize),
		microbatch.WithTimeout(batchDuration),
	)
}

func (s *TimedBatcherTestSuite) TestCallAfterStop() {
	// given
	s.Processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(batchSize))).Twice()

	var timedSend func(*bool)
	newTimer := func(_ time.Duration, f func(*bool)) timer.Timer {
		timedSend = f

		return s.Timer
	}

	s.NewTimer.EXPECT().New(batchDuration, mock.Anything).RunAndReturn(newTimer).Twice()
	s.Timer.EXPECT().Stop().Twice()

	// when
	for i := 0; i < 2*batchSize; i++ {
		_ = s.Batcher.Submit(i + 1)
		switch i {
		case batchSize - 1, 2*batchSize - 1:
			sent := true
			timedSend(&sent)
		}
	}

	s.Batcher.Send()

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func (s *TimedBatcherTestSuite) TestBatcherWithTimeouts() {
	// given
	s.Processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(2))).Twice()
	s.Processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(1))).Once()

	var timedSend func(*bool)
	newTimer := func(_ time.Duration, f func(*bool)) timer.Timer {
		timedSend = f

		return s.Timer
	}

	s.NewTimer.EXPECT().New(batchDuration, mock.Anything).RunAndReturn(newTimer).Times(3)
	s.Timer.EXPECT().Stop().Once()

	// when
	for i := 0; i < 5; i++ {
		_ = s.Batcher.Submit(i + 1)
		switch i {
		case 1, 3:
			sent := false
			timedSend(&sent)
		}
	}

	s.Batcher.Send()

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func hasRequestLen(n int) func([]internal.BatchRequest[int, string]) bool {
	return func(requests []internal.BatchRequest[int, string]) bool {
		return len(requests) == n
	}
}
