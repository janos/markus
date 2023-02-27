// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package indexmap

import (
	"bufio"
	"fmt"
	"os"
	"sort"
)

type Map struct {
	path  string
	free  []uint64
	index map[uint64]*uint64
}

func New(path string) (*Map, error) {
	m := &Map{
		path:  path,
		index: make(map[uint64]*uint64),
		free:  make([]uint64, 0),
	}
	return m, m.read()
}

func (m *Map) Get(index uint64) (realIndex uint64, has bool) {
	replacement, ok := m.index[index]
	if !ok {
		return index, true
	}
	if replacement == nil {
		return index, false // removed
	}
	return *replacement, true
}

func (m *Map) Add(index uint64) {
	if len(m.free) == 0 {
		return
	}

	replacement, ok := m.index[index]
	if ok && replacement != nil {
		return // exists replaced
	}
	free := m.free[0]
	if free == index {
		m.free = m.free[1:]
		delete(m.index, index)
		return
	}
	m.free = m.free[1:]
	m.index[index] = &free
}

func (m *Map) Remove(index uint64) {
	replacement, ok := m.index[index]
	if !ok {
		m.index[index] = nil
		m.free = append(m.free, index)
	}
	if replacement != nil {
		m.index[*replacement] = nil
		m.free = append(m.free, *replacement)
		delete(m.index, index)
	}
	sort.Slice(m.free, func(i, j int) bool {
		return m.free[i] < m.free[j]
	})
}

func (m *Map) FreeCount() int {
	return len(m.free)
}

func (m *Map) read() error {
	f, err := os.OpenFile(m.path, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	m.free = m.free[:0]
	for k := range m.index {
		delete(m.index, k)
	}

	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		text := s.Text()
		if text == "# replaced" || text == "# free" || text == "# removed" {
			break
		}
	}
	for s.Scan() {
		text := s.Text()
		if text == "# free" || text == "# removed" {
			break
		}
		var index uint64
		var replacement uint64
		if _, err := fmt.Sscan(text, &index, &replacement); err != nil {
			m.index = nil
			m.free = nil
			return fmt.Errorf("scan replaced index: %w", err)
		}
		m.index[index] = &replacement
	}
	for s.Scan() {
		text := s.Text()
		if text == "# free" {
			break
		}
		var index uint64
		if _, err := fmt.Sscan(text, &index); err != nil {
			m.index = nil
			m.free = nil
			return fmt.Errorf("scan replaced index: %w", err)
		}
		m.index[index] = nil
	}
	for s.Scan() {
		text := s.Text()
		var index uint64
		if _, err := fmt.Sscan(text, &index); err != nil {
			m.index = nil
			m.free = nil
			return fmt.Errorf("scan free index: %w", err)
		}
		m.free = append(m.free, index)
		m.index[index] = nil
	}

	return nil
}

func (m *Map) Write() error {
	f, err := os.OpenFile(m.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("# replaced\n"); err != nil {
		return fmt.Errorf("write removed header: %w", err)
	}

	for index, replacement := range m.index {
		if replacement == nil {
			continue
		}
		if _, err := f.WriteString(fmt.Sprintf("%v %v\n", index, *replacement)); err != nil {
			return fmt.Errorf("write replaced index: %w", err)
		}
	}

	if _, err := f.WriteString("# removed\n"); err != nil {
		return fmt.Errorf("write removed header: %w", err)
	}

	for index, replacement := range m.index {
		if replacement != nil {
			continue
		}
		if _, err := f.WriteString(fmt.Sprintf("%v\n", index)); err != nil {
			return fmt.Errorf("write replaced index: %w", err)
		}
	}

	if _, err := f.WriteString("# free\n"); err != nil {
		return fmt.Errorf("write free header: %w", err)
	}

	for _, index := range m.free {
		if _, err := f.WriteString(fmt.Sprintf("%v\n", index)); err != nil {
			return fmt.Errorf("write free index: %w", err)
		}
	}

	return nil
}
