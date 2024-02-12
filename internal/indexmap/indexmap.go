// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package indexmap

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type Map struct {
	path    string
	choices []uint64
	index   map[uint64]uint64
	free    []uint64
	cursor  uint64
}

func New(path string) (*Map, error) {
	m := &Map{
		path:    path,
		choices: make([]uint64, 0),       // physical -> virtual
		index:   make(map[uint64]uint64), // virtual -> physical
		free:    make([]uint64, 0),
	}
	return m, m.read()
}

func (m *Map) GetPhysical(index uint64) (physicalIndex uint64, has bool) {
	physicalIndex, ok := m.index[index]
	return physicalIndex, ok
}

func (m *Map) GetVirtual(physicalIndex uint64) (index uint64, has bool) {
	if physicalIndex >= uint64(len(m.choices)) {
		return 0, false
	}
	for _, f := range m.free {
		if f == physicalIndex {
			return 0, false
		}
	}
	return m.choices[physicalIndex], true
}

func (m *Map) Add() (physical, virtual uint64) {
	defer func() { m.cursor++ }()

	newVirtual := m.cursor

	if len(m.free) == 0 {
		m.choices = append(m.choices, newVirtual)
		physical = uint64(len(m.choices) - 1)
		m.index[newVirtual] = uint64(len(m.choices) - 1)
		return physical, newVirtual
	}

	physical = m.free[0]
	m.free = m.free[1:]
	m.choices[physical] = newVirtual
	m.index[newVirtual] = physical

	return physical, newVirtual
}

func (m *Map) Remove(index uint64) {
	location, ok := m.index[index]
	if !ok {
		return
	}

	m.free = append(m.free, location)
	delete(m.index, index)

	sort.Slice(m.free, func(i, j int) bool {
		return m.free[i] < m.free[j]
	})
}

func (m *Map) FreeCount() int {
	return len(m.free)
}

func (m *Map) Cursor() uint64 {
	return m.cursor
}

type store struct {
	Choices []uint64 `json:"choices,omitempty"`
	Free    []uint64 `json:"free,omitempty"`
	Cursor  uint64   `json:"cursor,omitempty"`
}

func (m *Map) read() error {
	f, err := os.OpenFile(m.path, os.O_RDONLY, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var s store
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return err
	}

	m.choices = s.Choices
	m.free = s.Free
	m.index = make(map[uint64]uint64)
	m.cursor = s.Cursor

Loop:
	for location, index := range m.choices {
		for _, f := range m.free {
			if f == uint64(location) {
				continue Loop
			}
		}
		m.index[index] = uint64(location)
	}

	return nil
}

func (m *Map) Write() error {
	f, err := os.OpenFile(m.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(store{
		Choices: m.choices,
		Free:    m.free,
		Cursor:  m.cursor,
	})
}
