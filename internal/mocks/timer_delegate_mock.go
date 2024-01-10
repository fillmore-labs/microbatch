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

// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// MockTimerDelegate is an autogenerated mock type for the TimerDelegate type
type MockTimerDelegate struct {
	mock.Mock
}

type MockTimerDelegate_Expecter struct {
	mock *mock.Mock
}

func (_m *MockTimerDelegate) EXPECT() *MockTimerDelegate_Expecter {
	return &MockTimerDelegate_Expecter{mock: &_m.Mock}
}

// Reset provides a mock function with given fields: d
func (_m *MockTimerDelegate) Reset(d time.Duration) bool {
	ret := _m.Called(d)

	if len(ret) == 0 {
		panic("no return value specified for Reset")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func(time.Duration) bool); ok {
		r0 = rf(d)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// MockTimerDelegate_Reset_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Reset'
type MockTimerDelegate_Reset_Call struct {
	*mock.Call
}

// Reset is a helper method to define mock.On call
//   - d time.Duration
func (_e *MockTimerDelegate_Expecter) Reset(d interface{}) *MockTimerDelegate_Reset_Call {
	return &MockTimerDelegate_Reset_Call{Call: _e.mock.On("Reset", d)}
}

func (_c *MockTimerDelegate_Reset_Call) Run(run func(d time.Duration)) *MockTimerDelegate_Reset_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(time.Duration))
	})
	return _c
}

func (_c *MockTimerDelegate_Reset_Call) Return(_a0 bool) *MockTimerDelegate_Reset_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTimerDelegate_Reset_Call) RunAndReturn(run func(time.Duration) bool) *MockTimerDelegate_Reset_Call {
	_c.Call.Return(run)
	return _c
}

// Stop provides a mock function with given fields:
func (_m *MockTimerDelegate) Stop() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Stop")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// MockTimerDelegate_Stop_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Stop'
type MockTimerDelegate_Stop_Call struct {
	*mock.Call
}

// Stop is a helper method to define mock.On call
func (_e *MockTimerDelegate_Expecter) Stop() *MockTimerDelegate_Stop_Call {
	return &MockTimerDelegate_Stop_Call{Call: _e.mock.On("Stop")}
}

func (_c *MockTimerDelegate_Stop_Call) Run(run func()) *MockTimerDelegate_Stop_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockTimerDelegate_Stop_Call) Return(_a0 bool) *MockTimerDelegate_Stop_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTimerDelegate_Stop_Call) RunAndReturn(run func() bool) *MockTimerDelegate_Stop_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockTimerDelegate creates a new instance of MockTimerDelegate. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTimerDelegate(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTimerDelegate {
	mock := &MockTimerDelegate{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
