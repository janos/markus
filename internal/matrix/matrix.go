// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"unsafe"
)

const (
	versionSize    = 8
	maxElementSize = 8
)

type Type interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

type Matrix[T Type] struct {
	mmap        *mmap
	dataPtr     unsafe.Pointer
	file        *os.File
	path        string
	elementSize int64

	decode func([]byte) T
	encode func(T) []byte
}

func New[T Type](path string) (*Matrix[T], error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	mmap, err := newMmap(f)
	if err != nil {
		return nil, fmt.Errorf("mmap file: %w", err)
	}

	elementSize := int64(unsafe.Sizeof(T(0)))

	m := &Matrix[T]{
		mmap:        mmap,
		file:        f,
		path:        path,
		elementSize: elementSize,
	}

	if len(*mmap) != 0 {
		m.dataPtr = unsafe.Add(unsafe.Pointer(&(*mmap)[0]), versionSize)
	}

	switch any(T(0)).(type) {
	case uint8:
		m.decode = func(data []byte) T { return (T)(data[0]) }
		m.encode = func(v T) []byte { return []byte{byte(v)} }
	case uint16:
		m.decode = func(data []byte) T { return (T)(binary.LittleEndian.Uint16(data)) }
		buf := make([]byte, elementSize)
		m.encode = func(v T) []byte { binary.LittleEndian.PutUint16(buf, uint16(v)); return buf }
	case uint32:
		m.decode = func(data []byte) T { return (T)(binary.LittleEndian.Uint32(data)) }
		buf := make([]byte, elementSize)
		m.encode = func(v T) []byte { binary.LittleEndian.PutUint32(buf, uint32(v)); return buf }
	case uint64:
		m.decode = func(data []byte) T { return (T)(binary.LittleEndian.Uint64(data)) }
		buf := make([]byte, elementSize)
		m.encode = func(v T) []byte { binary.LittleEndian.PutUint64(buf, uint64(v)); return buf }
	default:
		return nil, errors.New("unsupported type")
	}

	return m, nil
}

func (m *Matrix[T]) Version() int64 {
	if len(*m.mmap) < versionSize {
		return 0
	}
	return int64(binary.LittleEndian.Uint64((*m.mmap)[:versionSize]))
}

func (m *Matrix[T]) SetVersion(version int64) error {
	if len(*m.mmap) < versionSize {
		if _, _, err := m.Resize(0); err != nil {
			return err
		}
	}
	binary.LittleEndian.PutUint64((*m.mmap)[:versionSize], uint64(version))
	return nil
}

func (m *Matrix[T]) Resize(diff int64) (from, to int64, err error) {
	currentSize := m.Size()

	newSize := currentSize + diff

	if newSize < 0 {
		return 0, 0, fmt.Errorf("new size is negative %d (from %d)", newSize, currentSize)
	}

	if currentSize > 0 {
		if err := m.mmap.Sync(); err != nil {
			return 0, 0, fmt.Errorf("sync mmap: %w", err)
		}
	}

	if err := m.mmap.Unmap(); err != nil {
		return 0, 0, fmt.Errorf("unmap file: %w", err)
	}

	if err := m.file.Sync(); err != nil {
		return 0, 0, fmt.Errorf("sync file: %w", err)
	}

	fileSize := versionSize + newSize*newSize*m.elementSize
	if err := m.file.Truncate(fileSize); err != nil {
		return 0, 0, fmt.Errorf("truncate file: %w", err)
	}

	m.mmap, err = newMmap(m.file)
	if err != nil {
		return 0, 0, fmt.Errorf("mmap file: %w", err)
	}
	m.dataPtr = unsafe.Add(unsafe.Pointer(&(*m.mmap)[0]), versionSize)

	return currentSize, newSize, nil
}

func (m *Matrix[T]) Get(i, j int64) T {
	if m.dataPtr == nil {
		return T(0)
	}
	l := location(i, j)
	buf := (*[maxElementSize]byte)(unsafe.Add(m.dataPtr, l*m.elementSize))[0:m.elementSize:m.elementSize]
	return m.decode(buf)
}

func (m *Matrix[T]) Set(i, j int64, e T) {
	if m.dataPtr == nil {
		return
	}
	l := location(i, j)
	buf := m.encode(e)
	copy((*[maxElementSize]byte)(unsafe.Add(m.dataPtr, l*m.elementSize))[0:m.elementSize:m.elementSize], buf)
}

func (m *Matrix[T]) Sync() error {
	if err := m.mmap.Sync(); err != nil {
		return fmt.Errorf("sync mmap size %d: %w", len(*m.mmap), err)
	}
	if err := m.file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	return nil
}

func (m *Matrix[T]) Size() int64 {
	length := int64(len(*m.mmap))
	return floorSqrt((length - versionSize) / m.elementSize)
}

func (m *Matrix[T]) Close() error {
	if err := m.mmap.Unmap(); err != nil {
		return fmt.Errorf("unmap: %w", err)
	}
	if err := m.file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}

// matrix i,j = 0,0 0,1 1,0 1,1 0,2 1,2 2,0 2,1 2,2 0,3 1,3 2,3 3,0 3,1 3,2 3,3
// array  n   =   0   1   2   3   4   5   6   7   8   9  10  11  12  13  14  15
func location(i, j int64) int64 {
	if i < j {
		return j*j + i
	}
	return i*i + i + j
}

func floorSqrt(x int64) int64 {
	if x == 0 || x == 1 {
		return x
	}
	var start int64 = 1
	end := x / 2
	var ans int64
	for start <= end {
		mid := (start + end) / 2
		if mid*mid == x {
			return mid
		}
		if mid*mid < x {
			start = mid + 1
			ans = mid
		} else {
			end = mid - 1
		}
	}
	return ans
}
