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

package dataloader

import (
	"sync"

	"fillmore-labs.com/microbatch"
	"fillmore-labs.com/promise"
)

// DataLoader demonstrates how to use [microbatch.Batcher] to implement a simple Facebook [DataLoader].
// K and R define the key and result types for batching.
//
// [DataLoader]: https://www.youtube.com/watch?v=OQTnXNCDywA
type DataLoader[K comparable, R any] struct {
	mu      sync.RWMutex
	cache   map[K]*promise.Memoizer[R] // cache stores keys mapped to results
	batcher *microbatch.Batcher[K, R]  // batcher batches keys and retrieves results
}

// NewDataLoader create a new [DataLoader].
func NewDataLoader[K comparable, R any, KK ~[]K, RR ~[]R](
	processJobs func(keys KK) (RR, error), // processJobs retrieve results for the given keys
	correlate func(result R) K, // correlate maps each result back to its key
	opts ...microbatch.Option, // opts allows configuring the underlying [Batcher]
) *DataLoader[K, R] {
	batcher := microbatch.NewBatcher(
		processJobs,
		func(k K) K { return k },
		correlate,
		opts...,
	)

	return &DataLoader[K, R]{
		cache:   make(map[K]*promise.Memoizer[R]),
		batcher: batcher,
	}
}

// Load retrieves a value from the cache or loads it asynchronously.
func (d *DataLoader[K, R]) Load(key K) *promise.Memoizer[R] {
	d.mu.RLock()
	memoizer, ok := d.cache[key]
	d.mu.RUnlock()
	if ok {
		return memoizer
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	memoizer, ok = d.cache[key]
	if !ok {
		memoizer = d.batcher.Submit(key).Memoize()
		d.cache[key] = memoizer
	}

	return memoizer
}

// Send loads all submitted keys.
func (d *DataLoader[K, R]) Send() {
	d.batcher.Send()
}
