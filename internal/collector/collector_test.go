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
)

const settleGoRoutines = 10 * time.Millisecond

func TestCollectorTerminates(t *testing.T) {
	t.Parallel()

	// given
	processor := mocks.NewMockProcessor[int, string](t)

	requests := make(chan internal.BatchRequest[int, string])
	terminating := make(chan struct{})
	terminated := make(chan struct{})

	c := collector.Collector[int, string]{
		Requests:    requests,
		Terminating: terminating,
		Terminated:  terminated,
		Processor:   processor,
	}

	batcherDone := make(chan struct{})
	go func() {
		c.Run()
		close(batcherDone)
	}()

	// when
	terminating <- struct{}{}
	<-terminated

	// then
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	select {
	case <-batcherDone:
		t.Log("Collector shut down")

	case <-ctx.Done():
		t.Error("Collector did not shut down")
	}

	processor.AssertNotCalled(t, "Process", mock.Anything)
}

func TestCollector(t *testing.T) {
	t.Parallel()

	// given
	const batchSize = 5
	const batchDuration = 1 * time.Second

	processor := mocks.NewMockProcessor[int, string](t)
	processor.EXPECT().
		Process(mock.MatchedBy(hasRequestLen(batchSize))).
		Twice()

	tm := make(chan time.Time)
	d := mocks.NewMockTimerDelegate(t)
	d.EXPECT().Reset(batchDuration).Return(false).Twice()
	d.EXPECT().Stop().Return(false).Twice()

	requests := make(chan internal.BatchRequest[int, string])
	terminating := make(chan struct{})
	terminated := make(chan struct{})

	c := collector.Collector[int, string]{
		Requests:      requests,
		Terminating:   terminating,
		Terminated:    terminated,
		Processor:     processor,
		BatchSize:     batchSize,
		BatchDuration: batchDuration,
		Timer: &collector.Timer{
			C:     tm,
			Timer: d,
		},
	}

	go c.Run()

	// when
	for i := 0; i < batchSize; i++ {
		requests <- internal.BatchRequest[int, string]{Request: i + 1}
	}
	tm <- time.Time{}

	for i := batchSize; i < 2*batchSize; i++ {
		requests <- internal.BatchRequest[int, string]{Request: i + 1}
	}
	tm <- time.Time{}

	terminating <- struct{}{}
	<-terminated

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func TestCollectorWithTimeouts(t *testing.T) {
	t.Parallel()

	// given
	const batchDuration = 1 * time.Second

	processor := mocks.NewMockProcessor[int, string](t)
	processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(2))).Twice()
	processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(1))).Once()

	tm := make(chan time.Time)
	d := mocks.NewMockTimerDelegate(t)
	d.EXPECT().Reset(batchDuration).Return(false).Times(3)
	d.EXPECT().Stop().Return(true).Once()

	requests := make(chan internal.BatchRequest[int, string])
	terminating := make(chan struct{})
	terminated := make(chan struct{})

	c := collector.Collector[int, string]{
		Requests:      requests,
		Terminating:   terminating,
		Terminated:    terminated,
		Processor:     processor,
		BatchDuration: batchDuration,
		Timer: &collector.Timer{
			C:     tm,
			Timer: d,
		},
	}

	go c.Run()

	// when
	for i := 0; i < 2; i++ {
		requests <- internal.BatchRequest[int, string]{Request: i + 1}
	}
	tm <- time.Time{}
	for i := 2; i < 4; i++ {
		requests <- internal.BatchRequest[int, string]{Request: i + 1}
	}
	tm <- time.Time{}
	for i := 4; i < 5; i++ {
		requests <- internal.BatchRequest[int, string]{Request: i + 1}
	}
	terminating <- struct{}{}
	<-terminated

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func TestCollectorWithoutSize(t *testing.T) {
	t.Parallel()

	// given
	const batchDuration = 1 * time.Second

	processor := mocks.NewMockProcessor[int, string](t)
	processor.EXPECT().
		Process(mock.MatchedBy(hasRequestLen(1))).
		Twice()

	tm := make(chan time.Time)
	d := mocks.NewMockTimerDelegate(t)
	d.EXPECT().Reset(batchDuration).Return(true).Twice()
	d.EXPECT().Stop().Return(true).Once()

	requests := make(chan internal.BatchRequest[int, string])
	terminating := make(chan struct{})
	terminated := make(chan struct{})

	c := collector.Collector[int, string]{
		Requests:      requests,
		Terminating:   terminating,
		Terminated:    terminated,
		Processor:     processor,
		BatchDuration: batchDuration,
		Timer: &collector.Timer{
			C:     tm,
			Timer: d,
		},
	}

	go c.Run()

	// when
	requests <- internal.BatchRequest[int, string]{Request: 1}

	tm <- time.Time{}

	requests <- internal.BatchRequest[int, string]{Request: 2}

	terminating <- struct{}{}
	<-terminated

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func TestCollectorWithRealTimer(t *testing.T) {
	t.Parallel()

	// given
	const batchDuration = 1 * time.Nanosecond

	processor := mocks.NewMockProcessor[int, string](t)
	processor.EXPECT().Process(mock.MatchedBy(hasRequestLen(1))).Twice()

	requests := make(chan internal.BatchRequest[int, string])
	terminating := make(chan struct{})
	terminated := make(chan struct{})

	c := collector.Collector[int, string]{
		Requests:      requests,
		Terminating:   terminating,
		Terminated:    terminated,
		Processor:     processor,
		BatchDuration: batchDuration,
	}

	go c.Run()

	// when
	requests <- internal.BatchRequest[int, string]{Request: 1}

	time.Sleep(10 * time.Microsecond)

	requests <- internal.BatchRequest[int, string]{Request: 2}

	terminating <- struct{}{}
	<-terminated

	// then
	time.Sleep(settleGoRoutines)
	// assert expectations
}

func hasRequestLen(n int) func([]internal.BatchRequest[int, string]) bool {
	return func(requests []internal.BatchRequest[int, string]) bool {
		return len(requests) == n
	}
}
