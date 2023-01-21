// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset

import (
	"math/rand"
	"testing"
	"time"
)

func TestBitSet(t *testing.T) {
	contains := func(i int, values []int) bool {
		for _, v := range values {
			if i == v {
				return true
			}
		}
		return false
	}

	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	size := r.Intn(12345)
	s := New(size)
	values := make([]int, 0)
	for i, count := uint(0), uint(r.Intn(100)); i < count; i++ {
		values = append(values, r.Intn(size))
	}
	for _, v := range values {
		s.Set(v)
	}
	for i := 0; i < size; i++ {
		if contains(i, values) {
			if !s.IsSet(i) {
				t.Errorf("expected value %v is not set (seed %v)", i, seed)
			}
		} else {
			if s.IsSet(i) {
				t.Errorf("value %v is unexpectedly set (seed %v)", i, seed)
			}
		}
	}
}
