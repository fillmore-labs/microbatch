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

package collector_test

import (
	"context"
	"testing"
	"time"

	"fillmore-labs.com/microbatch/internal/collector"
	"fillmore-labs.com/microbatch/internal/mocks"
	internal "fillmore-labs.com/microbatch/internal/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const settleGoRoutines = 10 * time.Millisecond

type CollectorTestSuite struct {
	suite.Suite
	Processor   *mocks.MockProcessor[int, string]
	Requests    chan<- internal.BatchRequest[int, string]
	Terminating chan<- struct{}
	Terminated  <-chan struct{}
	Collector   *collector.Collector[int, string]

	C             chan<- time.Time
	TimerDelegate *mocks.MockTimerDelegate
}

func TestCollectorTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CollectorTestSuite))
}

func (s *CollectorTestSuite) SetupTest() {
	processor := mocks.NewMockProcessor[int, string](s.T())

	requests := make(chan internal.BatchRequest[int, string])
	terminating := make(chan struct{})
	terminated := make(chan struct{})

	s.Processor = processor
	s.Requests = requests
	s.Terminating = terminating
	s.Terminated = terminated

	tm := make(chan time.Time)
	d := mocks.NewMockTimerDelegate(s.T())

	s.C = tm
	s.TimerDelegate = d

	s.Collector = &collector.Collector[int, string]{
		Requests:    requests,
		Terminating: terminating,
		Terminated:  terminated,
		Processor:   processor,
		Timer:       &collector.Timer{C: tm, Delegate: d},
	}
}

func (s *CollectorTestSuite) terminateCollector() {
	s.Terminating <- struct{}{}
	<-s.Terminated
}

func (s *CollectorTestSuite) TestCollectorTerminates() {
	// given
	s.Collector.Timer = nil

	batcherDone := make(chan struct{})

	// when
	go func() {
		s.Collector.Run()
		close(batcherDone)
	}()

	s.terminateCollector()

	// then
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
		s.Fail("Collector did not shut down")

	case <-batcherDone:
	}

	s.Processor.AssertNotCalled(s.T(), "Process", mock.Anything)
}

func (s *CollectorTestSuite) TestCollector() {
	// given
	const batchSize = 5
	const batchDuration = 1 * time.Second

	s.Processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(batchSize))).Twice()

	s.TimerDelegate.EXPECT().Reset(batchDuration).Return(false).Twice()
	s.TimerDelegate.EXPECT().Stop().Return(false).Twice()

	s.Collector.BatchSize = batchSize
	s.Collector.BatchDuration = batchDuration

	// when
	go s.Collector.Run()

	for i := 0; i < 2*batchSize; i++ {
		s.Requests <- internal.BatchRequest[int, string]{Request: i + 1}
		switch i {
		case batchSize - 1, 2*batchSize - 1:
			s.C <- time.Time{}
		}
	}

	s.terminateCollector()

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func (s *CollectorTestSuite) TestCollectorWithTimeouts() {
	// given
	const batchDuration = 1 * time.Second

	s.Processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(2))).Twice()
	s.Processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(1))).Once()

	s.TimerDelegate.EXPECT().Reset(batchDuration).Return(false).Times(3)
	s.TimerDelegate.EXPECT().Stop().Return(true).Once()

	s.Collector.BatchDuration = batchDuration

	// when
	go s.Collector.Run()

	for i := 0; i < 5; i++ {
		s.Requests <- internal.BatchRequest[int, string]{Request: i + 1}
		switch i {
		case 1, 3:
			s.C <- time.Time{}
		}
	}

	s.terminateCollector()

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func (s *CollectorTestSuite) TestCollectorWithoutSize() {
	// given
	const batchDuration = 1 * time.Second

	s.Processor.EXPECT().
		Process(mock.MatchedBy(hasRequestLen(1))).
		Twice()

	s.TimerDelegate.EXPECT().Reset(batchDuration).Return(true).Twice()
	s.TimerDelegate.EXPECT().Stop().Return(true).Once()

	s.Collector.BatchDuration = batchDuration

	// when
	go s.Collector.Run()

	s.Requests <- internal.BatchRequest[int, string]{Request: 1}

	s.C <- time.Time{}

	s.Requests <- internal.BatchRequest[int, string]{Request: 2}

	s.terminateCollector()

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func (s *CollectorTestSuite) TestCollectorWithRealTimer() {
	// given
	const batchDuration = 1 * time.Nanosecond

	s.Processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(1))).Twice()

	s.Collector.BatchDuration = batchDuration
	s.Collector.Timer = nil

	// when
	go s.Collector.Run()

	s.Requests <- internal.BatchRequest[int, string]{Request: 1}

	time.Sleep(10 * time.Microsecond)

	s.Requests <- internal.BatchRequest[int, string]{Request: 2}

	s.terminateCollector()

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func hasRequestLen(n int) func([]internal.BatchRequest[int, string]) bool {
	return func(requests []internal.BatchRequest[int, string]) bool {
		return len(requests) == n
	}
}
