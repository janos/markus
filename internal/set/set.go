// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package set

import (
	"bufio"
	"fmt"
	"os"
)

type Set[C comparable] struct {
	path string
	set  map[C]struct{}
}

func New[C comparable](path string) (*Set[C], error) {
	l := &Set[C]{
		path: path,
		set:  make(map[C]struct{}),
	}
	if err := l.read(); err != nil {
		return nil, fmt.Errorf("read set: %w", err)
	}
	return l, nil
}

func (l *Set[C]) Add(items ...C) {
	for _, item := range items {
		l.set[item] = struct{}{}
	}
}

func (l *Set[C]) Set() map[C]struct{} {
	return l.set
}

func (l *Set[C]) Has(item C) bool {
	_, ok := l.set[item]
	return ok
}

func (l *Set[C]) Clear() error {
	for k := range l.set {
		delete(l.set, k)
	}
	return l.Sync()
}

func (l *Set[C]) Size() int {
	return len(l.set)
}

func (l *Set[C]) read() error {
	f, err := os.OpenFile(l.path, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	for k := range l.set {
		delete(l.set, k)
	}

	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		var item C
		if _, err := fmt.Sscan(s.Text(), &item); err != nil {
			return fmt.Errorf("scan item: %w", err)
		}
		l.set[item] = struct{}{}
	}

	return nil
}

func (l *Set[C]) Sync() error {
	f, err := os.OpenFile(l.path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("truncate file: %w", err)
	}

	for _, item := range l.set {
		if _, err := f.WriteString(fmt.Sprintf("%v\n", item)); err != nil {
			return fmt.Errorf("write item: %w", err)
		}
	}

	return nil
}
