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

	assertMap(t, m, nil, 0, 0)

	m.Remove(10)

	assertMap(t, m, nil, 0, 0)

	physical, virtual := m.Add()
	assertEqual(t, "physical", physical, 0)
	assertEqual(t, "virtual", virtual, 0)

	assertMap(t, m, map[uint64]uint64{0: 0}, 1, 0)

	physical, virtual = m.Add()
	assertEqual(t, "physical", physical, 1)
	assertEqual(t, "virtual", virtual, 1)

	assertMap(t, m, map[uint64]uint64{0: 0, 1: 1}, 2, 0)

	physical, virtual = m.Add()
	assertEqual(t, "physical", physical, 2)
	assertEqual(t, "virtual", virtual, 2)

	assertMap(t, m, map[uint64]uint64{0: 0, 1: 1, 2: 2}, 3, 0)

	m.Remove(1)

	assertMap(t, m, map[uint64]uint64{0: 0, 2: 2}, 3, 1)

	physical, virtual = m.Add()
	assertEqual(t, "physical", physical, 1)
	assertEqual(t, "virtual", virtual, 3)

	assertMap(t, m, map[uint64]uint64{0: 0, 2: 2, 3: 1}, 4, 0)

	m.Remove(10)

	physical, virtual = m.Add()
	assertEqual(t, "physical", physical, 3)
	assertEqual(t, "virtual", virtual, 4)

	physical, virtual = m.Add()
	assertEqual(t, "physical", physical, 4)
	assertEqual(t, "virtual", virtual, 5)

	assertMap(t, m, map[uint64]uint64{0: 0, 2: 2, 3: 1, 4: 3, 5: 4}, 6, 0)

	m.Remove(0)
	m.Remove(3)

	assertMap(t, m, map[uint64]uint64{2: 2, 4: 3, 5: 4}, 6, 2)

	physical, virtual = m.Add()
	assertEqual(t, "physical", physical, 0)
	assertEqual(t, "virtual", virtual, 6)

	assertMap(t, m, map[uint64]uint64{6: 0, 2: 2, 4: 3, 5: 4}, 7, 1)

	physical, virtual = m.Add()
	assertEqual(t, "physical", physical, 1)
	assertEqual(t, "virtual", virtual, 7)

	assertMap(t, m, map[uint64]uint64{6: 0, 7: 1, 2: 2, 4: 3, 5: 4}, 8, 0)

	physical, virtual = m.Add()
	assertEqual(t, "physical", physical, 5)
	assertEqual(t, "virtual", virtual, 8)

	assertMap(t, m, map[uint64]uint64{6: 0, 7: 1, 2: 2, 4: 3, 5: 4, 8: 5}, 9, 0)

	physical, virtual = m.Add()
	assertEqual(t, "physical", physical, 6)
	assertEqual(t, "virtual", virtual, 9)

	assertMap(t, m, map[uint64]uint64{6: 0, 7: 1, 2: 2, 4: 3, 5: 4, 8: 5, 9: 6}, 10, 0)

	m.Remove(9)
	m.Remove(8)
	m.Remove(2)

	assertMap(t, m, map[uint64]uint64{6: 0, 7: 1, 4: 3, 5: 4}, 10, 3)

	assertError(t, m.Write(), nil)

	m, _ = newMap(t, dir)

	assertMap(t, m, map[uint64]uint64{6: 0, 7: 1, 4: 3, 5: 4}, 10, 3)
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

func assertMap(t testing.TB, m *indexmap.Map, indexes map[uint64]uint64, cursor uint64, freeCount int) {
	t.Helper()

	for i := uint64(0); i < 100; i++ {
		gotphysicalIndex, gotHas := m.GetPhysical(i)
		wantphysicalIndex, wantHas := indexes[i]
		assertEqual(t, fmt.Sprintf("index %v", i), gotphysicalIndex, wantphysicalIndex)
		assertEqual(t, fmt.Sprintf("has %v", i), gotHas, wantHas)
	}

	virtualIndexes := make(map[uint64]uint64)

	for k, v := range indexes {
		virtualIndexes[v] = k
	}

	for i := uint64(0); i < 100; i++ {
		gotVirtualIndex, gotHas := m.GetVirtual(i)
		wantVirtualIndex, wantHas := virtualIndexes[i]
		assertEqual(t, fmt.Sprintf("virtual index %v", i), gotVirtualIndex, wantVirtualIndex)
		assertEqual(t, fmt.Sprintf("virtual has %v", i), gotHas, wantHas)
	}

	assertEqual(t, "free count", m.FreeCount(), freeCount)
	assertEqual(t, "cursor", m.Cursor(), cursor)
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
