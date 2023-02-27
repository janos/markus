// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"bytes"
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
	elementSize uint64

	decode func([]byte) T
	encode func(T) []byte
}

func New[T Type](path string) (*Matrix[T], error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	if err := flock(f); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock file: %w", err)
	}

	mmap, err := newMmap(f)
	if err != nil {
		return nil, fmt.Errorf("mmap file: %w", err)
	}

	elementSize := uint64(unsafe.Sizeof(T(0)))

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

func (m *Matrix[T]) Resize(diff int64) (from, to uint64, err error) {
	currentSize := m.Size()

	var newSize uint64
	if diff > 0 {
		newSize = currentSize + uint64(diff)
	} else {
		newSize = currentSize - uint64(-diff)
	}

	// todo: detect overflow

	if currentSize > 0 {
		if err := m.mmap.sync(); err != nil {
			return 0, 0, fmt.Errorf("sync mmap: %w", err)
		}
	}

	if err := m.mmap.unmap(); err != nil {
		return 0, 0, fmt.Errorf("unmap file: %w", err)
	}

	if err := m.file.Sync(); err != nil {
		return 0, 0, fmt.Errorf("sync file: %w", err)
	}

	fileSize := int64(versionSize + newSize*newSize*m.elementSize)
	if err := m.file.Truncate(fileSize); err != nil {
		return 0, 0, fmt.Errorf("truncate file: %w", err)
	}

	m.mmap, err = newMmap(m.file)
	if err != nil {
		_ = m.file.Close()
		return 0, 0, fmt.Errorf("mmap file: %w", err)
	}

	m.dataPtr = unsafe.Add(unsafe.Pointer(&(*m.mmap)[0]), versionSize)

	return currentSize, newSize, nil
}

func (m *Matrix[T]) Get(i, j uint64) T {
	if m.dataPtr == nil {
		return T(0)
	}
	l := location(i, j)
	buf := (*[maxElementSize]byte)(unsafe.Add(m.dataPtr, l*m.elementSize))[0:m.elementSize:m.elementSize]
	return m.decode(buf)
}

func (m *Matrix[T]) Set(i, j uint64, e T) {
	if m.dataPtr == nil {
		return
	}
	l := location(i, j)
	buf := m.encode(e)
	src := (*[maxElementSize]byte)(unsafe.Add(m.dataPtr, l*m.elementSize))[0:m.elementSize:m.elementSize]
	if bytes.Equal(src, buf) {
		return
	}
	copy(src, buf)
}

func (m *Matrix[T]) Inc(i, j uint64) {
	if m.dataPtr == nil {
		return
	}
	l := location(i, j)
	buf := (*[maxElementSize]byte)(unsafe.Add(m.dataPtr, l*m.elementSize))[0:m.elementSize:m.elementSize]
	incrementLittleEndian(buf)
}

func (m *Matrix[T]) Dec(i, j uint64) {
	if m.dataPtr == nil {
		return
	}
	l := location(i, j)
	buf := (*[maxElementSize]byte)(unsafe.Add(m.dataPtr, l*m.elementSize))[0:m.elementSize:m.elementSize]
	decrementLittleEndian(buf)
}

func (m *Matrix[T]) Sync() error {
	if err := m.mmap.sync(); err != nil {
		return fmt.Errorf("sync mmap size %d: %w", len(*m.mmap), err)
	}
	if err := m.file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	return nil
}

func (m *Matrix[T]) Size() uint64 {
	length := uint64(len(*m.mmap))
	if length < versionSize {
		return 0
	}
	return floorSqrt((length - versionSize) / m.elementSize)
}

func (m *Matrix[T]) Close() error {
	if err := m.mmap.unmap(); err != nil {
		return fmt.Errorf("unmap: %w", err)
	}
	if err := funlock(m.file); err != nil {
		return fmt.Errorf("unlock file: %w", err)
	}
	if err := m.file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}

// matrix i,j = 0,0 0,1 1,0 1,1 0,2 1,2 2,0 2,1 2,2 0,3 1,3 2,3 3,0 3,1 3,2 3,3
// array  n   =   0   1   2   3   4   5   6   7   8   9  10  11  12  13  14  15
func location(i, j uint64) uint64 {
	if i < j {
		return j*j + i
	}
	return i*i + i + j
}

func floorSqrt(x uint64) uint64 {
	if x == 0 || x == 1 {
		return x
	}
	var start uint64 = 1
	end := x / 2
	var ans uint64
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

func incrementLittleEndian(input []uint8) {
	// Increment the least significant byte by 1
	input[0]++

	// Propagate any carries to more significant bytes
	for i, last := 0, len(input)-1; i < last; i++ {
		if input[i] == 0 {
			input[i+1]++
		} else {
			break
		}
	}
}

func decrementLittleEndian(input []uint8) {
	// Decrement the least significant byte by 1
	input[0]--

	// Propagate any borrows to more significant bytes
	for i, last := 0, len(input)-1; i < last; i++ {
		if input[i] == 0xFF {
			input[i+1]--
		} else {
			break
		}
	}
}
