// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package indexmap_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"resenje.org/markus/internal/indexmap"
)

func TestMap(t *testing.T) {
	m, dir := newMap(t, "")

	assertEqual(t, "free count", m.FreeCount(), 0)

	assertMap(t, m, nil, nil, 0)

	m.Remove(10)

	assertMap(t, m, nil, []uint64{10}, 1)

	m.Add(30)

	assertMap(t, m, map[uint64]uint64{30: 10}, []uint64{10}, 0)

	m.Add(40)

	assertMap(t, m, map[uint64]uint64{30: 10}, []uint64{10}, 0)

	m.Remove(30)

	assertMap(t, m, nil, []uint64{10}, 1)

	m.Remove(10)

	assertMap(t, m, nil, []uint64{10}, 1)

	m.Add(10)

	assertMap(t, m, nil, nil, 0)

	m.Add(30)

	assertMap(t, m, nil, nil, 0)

	m.Add(1)

	assertMap(t, m, nil, nil, 0)

	m.Remove(5)
	m.Remove(6)

	assertMap(t, m, nil, []uint64{5, 6}, 2)

	m.Add(6)
	m.Add(20)
	m.Add(22)
	m.Remove(8)

	assertMap(t, m, map[uint64]uint64{6: 5, 20: 6}, []uint64{5, 8}, 1)

	assertError(t, m.Write(), nil)

	m, _ = newMap(t, dir)

	assertMap(t, m, map[uint64]uint64{6: 5, 20: 6}, []uint64{5, 8}, 1)
}

func newMap(t testing.TB, dir string) (*indexmap.Map, string) {
	t.Helper()

	if dir == "" {
		dir = t.TempDir()
	}

	filename := filepath.Join(dir, "indexmap")

	m, err := indexmap.New(filename)
	assertError(t, err, nil)

	return m, dir
}

func assertMap(t testing.TB, m *indexmap.Map, realIndexes map[uint64]uint64, removed []uint64, freeCount int) {
	t.Helper()

	for i := uint64(0); i < 100; i++ {
		gotRealIndex, gotOK := m.Get(i)
		wantRealIndex, ok := realIndexes[i]
		if !ok {
			wantRealIndex = i
		}
		assertEqual(t, fmt.Sprintf("index %v", i), gotRealIndex, wantRealIndex)
		assertEqual(t, fmt.Sprintf("ok %v", i), gotOK, !has(removed, i))
	}

	assertEqual(t, "free count", m.FreeCount(), freeCount)
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
		t.Fatalf("got error %[1]T %[1]q, want %[2]T %[2]q", got, want)
	}
}

func has(l []uint64, e uint64) bool {
	for _, x := range l {
		if x == e {
			return true
		}
	}
	return false
}
