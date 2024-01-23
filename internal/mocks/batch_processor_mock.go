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

import mock "github.com/stretchr/testify/mock"

// MockBatchProcessor is an autogenerated mock type for the BatchProcessor type
type MockBatchProcessor[QQ interface{}, RR interface{}] struct {
	mock.Mock
}

type MockBatchProcessor_Expecter[QQ interface{}, RR interface{}] struct {
	mock *mock.Mock
}

func (_m *MockBatchProcessor[QQ, RR]) EXPECT() *MockBatchProcessor_Expecter[QQ, RR] {
	return &MockBatchProcessor_Expecter[QQ, RR]{mock: &_m.Mock}
}

// ProcessJobs provides a mock function with given fields: jobs
func (_m *MockBatchProcessor[QQ, RR]) ProcessJobs(jobs QQ) (RR, error) {
	ret := _m.Called(jobs)

	if len(ret) == 0 {
		panic("no return value specified for ProcessJobs")
	}

	var r0 RR
	var r1 error
	if rf, ok := ret.Get(0).(func(QQ) (RR, error)); ok {
		return rf(jobs)
	}
	if rf, ok := ret.Get(0).(func(QQ) RR); ok {
		r0 = rf(jobs)
	} else {
		r0 = ret.Get(0).(RR)
	}

	if rf, ok := ret.Get(1).(func(QQ) error); ok {
		r1 = rf(jobs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockBatchProcessor_ProcessJobs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ProcessJobs'
type MockBatchProcessor_ProcessJobs_Call[QQ interface{}, RR interface{}] struct {
	*mock.Call
}

// ProcessJobs is a helper method to define mock.On call
//   - jobs QQ
func (_e *MockBatchProcessor_Expecter[QQ, RR]) ProcessJobs(jobs interface{}) *MockBatchProcessor_ProcessJobs_Call[QQ, RR] {
	return &MockBatchProcessor_ProcessJobs_Call[QQ, RR]{Call: _e.mock.On("ProcessJobs", jobs)}
}

func (_c *MockBatchProcessor_ProcessJobs_Call[QQ, RR]) Run(run func(jobs QQ)) *MockBatchProcessor_ProcessJobs_Call[QQ, RR] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(QQ))
	})
	return _c
}

func (_c *MockBatchProcessor_ProcessJobs_Call[QQ, RR]) Return(_a0 RR, _a1 error) *MockBatchProcessor_ProcessJobs_Call[QQ, RR] {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockBatchProcessor_ProcessJobs_Call[QQ, RR]) RunAndReturn(run func(QQ) (RR, error)) *MockBatchProcessor_ProcessJobs_Call[QQ, RR] {
	_c.Call.Return(run)
	return _c
}

// NewMockBatchProcessor creates a new instance of MockBatchProcessor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockBatchProcessor[QQ interface{}, RR interface{}](t interface {
	mock.TestingT
	Cleanup(func())
}) *MockBatchProcessor[QQ, RR] {
	mock := &MockBatchProcessor[QQ, RR]{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
