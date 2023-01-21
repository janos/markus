// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset

type Type interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type BitSet[T Type] []T

func New[T Type](size T) BitSet[T] {
	return BitSet[T](make([]T, size/64+1))
}

func (s BitSet[T]) Set(i T) {
	s[i/64] |= 1 << (i % 64)
}

func (s BitSet[T]) IsSet(i T) bool {
	return s[i/64]&(1<<(i%64)) != 0
}
