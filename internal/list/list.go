// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package list

import (
	"bufio"
	"fmt"
	"os"
)

type List[C comparable] struct {
	path     string
	elements []C
	index    map[C]int64
}

func New[C comparable](path string) (*List[C], error) {
	l := &List[C]{
		path:  path,
		index: make(map[C]int64),
	}
	if err := l.read(); err != nil {
		return nil, fmt.Errorf("read list: %w", err)
	}
	return l, nil
}

func (l *List[C]) Append(items ...C) (int, error) {
	var count int
	for _, item := range items {
		if _, ok := l.index[item]; ok {
			continue
		}
		index := int64(len(l.elements))
		l.elements = append(l.elements, item)
		l.index[item] = index
		count++
	}
	if err := l.write(); err != nil {
		return 0, fmt.Errorf("write list: %w", err)
	}
	return count, nil
}

func (l *List[C]) Elements() []C {
	return l.elements
}

func (l *List[C]) Index(item C) (int64, bool) {
	index, ok := l.index[item]
	return index, ok
}

func (l *List[C]) Size() int {
	return len(l.elements)
}

func (l *List[C]) read() error {
	f, err := os.OpenFile(l.path, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	l.elements = l.elements[:0]
	for k := range l.index {
		delete(l.index, k)
	}

	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		var item C
		var index int64
		if _, err := fmt.Sscan(s.Text(), &index, &item); err != nil {
			l.elements = nil
			l.index = nil
			return fmt.Errorf("scan item: %w", err)
		}
		if le := int64(len(l.elements)); index >= le {
			l.elements = append(l.elements, make([]C, index-le+1)...)
		}
		l.elements[index] = item
		l.index[item] = index
	}

	return nil
}

func (l *List[C]) write() error {
	f, err := os.OpenFile(l.path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("truncate file: %w", err)
	}

	for i, item := range l.elements {
		if _, err := f.WriteString(fmt.Sprintf("%d %v\n", i, item)); err != nil {
			return fmt.Errorf("write item: %w", err)
		}
	}

	return nil
}
