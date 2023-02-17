// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset

import "math/bits"

type BitSet struct {
	bits      []uint64
	remainder int
}

func New(size uint64) BitSet {
	return BitSet{
		bits:      make([]uint64, size/64+1),
		remainder: int(size % 64),
	}
}

func (s BitSet) Set(i uint64) {
	s.bits[i/64] |= 1 << (i % 64)
}

func (s BitSet) IsSet(i uint64) bool {
	return s.bits[i/64]&(1<<(i%64)) != 0
}

func (s BitSet) IterateSet(callback func(uint64)) {
	var bitset uint64
	l := len(s.bits)
	for k := 0; k < l; k++ {
		bitset = s.bits[k]
		for bitset != 0 {
			t := bitset & -bitset
			r := bits.TrailingZeros64(bitset)
			callback(uint64(k*64 + r))
			bitset ^= t
		}
	}
}

func (s BitSet) IterateUnset(callback func(uint64)) {
	var bitset uint64
	l := len(s.bits)
	for k := 0; k < l-1; k++ {
		bitset = ^s.bits[k]
		for bitset != 0 {
			t := bitset & -bitset
			r := bits.TrailingZeros64(bitset)
			callback(uint64(k*64 + r))
			bitset ^= t
		}
	}
	k := l - 1
	bitset = ^s.bits[k]
	for bitset != 0 {
		t := bitset & -bitset
		r := bits.TrailingZeros64(bitset)
		if r >= s.remainder {
			break
		}
		callback(uint64(k*64 + r))
		bitset ^= t
	}
}
