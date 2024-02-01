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

	"resenje.org/markus/internal/matrix"
)

const matrixSize = 1000

func TestMatrix(t *testing.T) {
	m := newMatrix(t)

	elementValueFirst := func(i, j uint64) uint32 {
		return uint32((i+1)*100000000000000000 + j + 1)
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

	for i := uint64(0); i < size; i++ {
		for j := uint64(0); j < size; j++ {
			e := elementValueFirst(i, j)
			m.Set(i, j, e)
		}
	}

	for i := uint64(0); i < size; i++ {
		for j := uint64(0); j < size; j++ {
			e := m.Get(i, j)
			assertEqual(t, fmt.Sprintf("element (%d,%d)", i, j), e, elementValueFirst(i, j))
		}
	}

	elementValueSecond := func(i, j uint64) uint32 {
		return uint32((i+1)*111111111111111111 + j + 1)
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

	for i := uint64(0); i < size; i++ {
		for j := uint64(0); j < size; j++ {
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

	for i := uint64(0); i < to; i++ {
		for j := uint64(0); j < to; j++ {
			want := m.Get(i, j) + 1
			m.Inc(i, j)
			got := m.Get(i, j)
			assertEqual(t, fmt.Sprintf("element (%d,%d)", i, j), got, want)
		}
	}

	for i := uint64(0); i < to; i++ {
		for j := uint64(0); j < to; j++ {
			want := m.Get(i, j) - 1
			m.Dec(i, j)
			got := m.Get(i, j)
			assertEqual(t, fmt.Sprintf("element (%d,%d)", i, j), got, want)
		}
	}

	version := m.Version()
	assertEqual(t, "version", version, 0)

	err = m.SetVersion(123)
	assertError(t, err, nil)

	version = m.Version()
	assertEqual(t, "version", version, 123)

	{
		m := newMatrix(t)

		version := m.Version()
		assertEqual(t, "version", version, 0)

		err = m.SetVersion(123)
		assertError(t, err, nil)

	}
}

func BenchmarkMatrix_Get(b *testing.B) {
	m := newMatrix(b)

	const size = matrixSize

	from, to, err := m.Resize(size)
	assertError(b, err, nil)
	assertEqual(b, "from", from, 0)
	assertEqual(b, "to", to, size)

	b.ResetTimer()

	var e uint32
	for i := 0; i < b.N; i++ {
		i, j := rand.Uint64()%size, rand.Uint64()%size
		e = m.Get(i, j)
	}
	_ = e
}

func BenchmarkMatrix_Set(b *testing.B) {
	m := newMatrix(b)

	const size = matrixSize

	from, to, err := m.Resize(size)
	assertError(b, err, nil)
	assertEqual(b, "from", from, 0)
	assertEqual(b, "to", to, size)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		i, j := rand.Uint64()%size, rand.Uint64()%size
		m.Set(i, j, rand.Uint32())
	}

	b.StartTimer()

	err = m.Sync()
	assertError(b, err, nil)
}

func BenchmarkMatrix_Inc(b *testing.B) {
	m := newMatrix(b)

	const size = matrixSize

	from, to, err := m.Resize(size)
	assertError(b, err, nil)
	assertEqual(b, "from", from, 0)
	assertEqual(b, "to", to, size)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		i, j := rand.Uint64()%size, rand.Uint64()%size
		m.Inc(i, j)
	}

	b.StartTimer()

	err = m.Sync()
	assertError(b, err, nil)
}

func newMatrix(t testing.TB) *matrix.Matrix {
	t.Helper()

	dir := t.TempDir()

	m, err := matrix.New(filepath.Join(dir, "matrix.mx"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Sync(); err != nil {
			t.Error(err)
		}
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
