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
	contains := func(i uint64, values []uint64) bool {
		for _, v := range values {
			if i == v {
				return true
			}
		}
		return false
	}

	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	size := r.Uint64()%12345 + 1234
	s := New(size)
	values := make([]uint64, 0)
	for i, count := uint64(0), r.Uint64()%1000; i < count; i++ {
		v := r.Uint64() % size
		if contains(v, values) {
			continue
		}
		values = append(values, v)
	}
	missing := make([]uint64, 0)
	for i := uint64(0); i < size; i++ {
		if !contains(i, values) {
			missing = append(missing, i)
		}
	}
	for _, v := range values {
		s.Set(v)
	}
	for i := uint64(0); i < size; i++ {
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

	{
		var iterated int
		s.IterateSet(func(i uint64) {
			if !contains(i, values) {
				t.Errorf("value %v is not set (seed %v)", i, seed)
			}
			if contains(i, missing) {
				t.Errorf("value %v is unexpectedly set (seed %v)", i, seed)
			}
			iterated++
		})
		if iterated != len(values) {
			t.Errorf("iterated %v values, expected %v (seed %v)", iterated, len(values), seed)
		}
	}

	{
		var iterated int
		s.IterateUnset(func(i uint64) {
			if contains(i, values) {
				t.Errorf("value %v is unexpectedly set (seed %v)", i, seed)
			}
			if !contains(i, missing) {
				t.Errorf("value %v is not missing (seed %v)", i, seed)
			}
			if i != missing[iterated] {
				t.Errorf("iterated value %v, expected %v (seed %v)", i, missing[iterated], seed)
			}
			iterated++
		})
		if iterated != len(missing) {
			t.Errorf("iterated %v values, expected %v (seed %v)", iterated, len(missing), seed)
		}
	}
}
