// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix_test

import (
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"resenje.org/markus/internal/matrix"
)

const matrixSize = 1000

func TestMatrix(t *testing.T) {
	m := newMatrix[uint16](t)

	elementValueFirst := func(i, j int64) uint16 {
		return uint16((i+1)*100000000000000000 + j + 1)
	}

	from, to, err := m.Resize(1)
	assertError(t, err, nil)
	assertEqual(t, "from", from, 0)
	assertEqual(t, "to", to, 1)
	assertEqual(t, "size", to, m.Size())

	from, to, err = m.Resize(2)
	assertError(t, err, nil)
	assertEqual(t, "from", from, 1)
	assertEqual(t, "to", to, 3)
	assertEqual(t, "size", to, m.Size())

	from, to, err = m.Resize(1)
	assertError(t, err, nil)
	assertEqual(t, "from", from, 3)
	assertEqual(t, "to", to, 4)
	assertEqual(t, "size", to, m.Size())

	size := to

	for i := int64(0); i < size; i++ {
		for j := int64(0); j < size; j++ {
			e := elementValueFirst(i, j)
			m.Set(i, j, e)
		}
	}

	for i := int64(0); i < size; i++ {
		for j := int64(0); j < size; j++ {
			e := m.Get(i, j)
			assertEqual(t, fmt.Sprintf("element (%d,%d)", i, j), e, elementValueFirst(i, j))
		}
	}

	elementValueSecond := func(i, j int64) uint16 {
		return uint16((i+1)*111111111111111111 + j + 1)
	}

	from, to, err = m.Resize(3)
	assertError(t, err, nil)
	assertEqual(t, "from", from, size)
	assertEqual(t, "to", to, size+3)

	for i := size; i < size+3; i++ {
		for j := size; j < size+3; j++ {
			m.Set(i, j, elementValueSecond(i, j))
		}
	}

	for i := int64(0); i < size; i++ {
		for j := int64(0); j < size; j++ {
			e := m.Get(i, j)
			if i >= size || j >= size {
				assertEqual(t, fmt.Sprintf("element (%d,%d)", i, j), e, elementValueSecond(i, j))
			} else {
				assertEqual(t, fmt.Sprintf("element (%d,%d)", i, j), e, elementValueFirst(i, j))
			}
		}
	}

	from, to, err = m.Resize(-2)
	assertError(t, err, nil)
	assertEqual(t, "from", from, size+3)
	assertEqual(t, "to", to, size+1)
	assertEqual(t, "size", to, m.Size())
}

func BenchmarkMatrix_Get(b *testing.B) {
	m := newMatrix[uint64](b)

	const size = matrixSize

	from, to, err := m.Resize(size)
	assertError(b, err, nil)
	assertEqual(b, "from", from, 0)
	assertEqual(b, "to", to, size)

	rand.Seed(time.Now().UnixNano())

	b.ResetTimer()

	var e uint64
	for i := 0; i < b.N; i++ {
		i, j := rand.Int63n(size), rand.Int63n(size)
		e = m.Get(i, j)
	}
	_ = e
}

func BenchmarkMatrix_Set(b *testing.B) {
	m := newMatrix[uint64](b)

	const size = matrixSize

	from, to, err := m.Resize(size)
	assertError(b, err, nil)
	assertEqual(b, "from", from, 0)
	assertEqual(b, "to", to, size)

	rand.Seed(time.Now().UnixNano())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		i, j := rand.Int63n(size), rand.Int63n(size)
		m.Set(i, j, rand.Uint64())
	}

	b.StartTimer()

	err = m.Sync()
	assertError(b, err, nil)
}

func newMatrix[T matrix.Type](t testing.TB) *matrix.Matrix[T] {
	t.Helper()

	dir := t.TempDir()

	m, err := matrix.New[T](filepath.Join(dir, "matrix.mx"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	return m
}

func assertEqual[T any](t testing.TB, name string, got, want T) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s %+v, want %+v", name, got, want)
	}
}

func assertError(t testing.TB, got, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Errorf("got error %[1]T %[1]q, want %[2]T %[2]q", got, want)
	}
}
