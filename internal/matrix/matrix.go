// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"unsafe"
)

const (
	versionSize = 8
	elementSize = uint64(unsafe.Sizeof(uint32(0)))
)

type Matrix struct {
	mmap      *mmap
	dataPtr   unsafe.Pointer
	file      *os.File
	path      string
	encodeBuf []byte
}

func New(path string) (*Matrix, error) {
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

	m := &Matrix{
		mmap:      mmap,
		file:      f,
		path:      path,
		encodeBuf: make([]byte, elementSize),
	}

	if len(*mmap) != 0 {
		m.dataPtr = unsafe.Add(unsafe.Pointer(&(*mmap)[0]), versionSize)
	}
	return m, nil
}

func (m *Matrix) Version() int64 {
	if len(*m.mmap) < versionSize {
		return 0
	}
	return int64(binary.LittleEndian.Uint64((*m.mmap)[:versionSize]))
}

func (m *Matrix) SetVersion(version int64) error {
	if len(*m.mmap) < versionSize {
		if _, _, err := m.Resize(0); err != nil {
			return err
		}
	}
	binary.LittleEndian.PutUint64((*m.mmap)[:versionSize], uint64(version))
	return nil
}

func (m *Matrix) Resize(diff int64) (from, to uint64, err error) {
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

	fileSize := int64(versionSize + newSize*newSize*elementSize)
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

func (m *Matrix) Get(i, j uint64) uint32 {
	l := location(i, j)
	buf := (*[elementSize]byte)(unsafe.Add(m.dataPtr, l*elementSize))[0:elementSize:elementSize]
	return m.decode(buf)
}

func (m *Matrix) Set(i, j uint64, e uint32) {
	l := location(i, j)
	buf := m.encode(e)
	src := (*[elementSize]byte)(unsafe.Add(m.dataPtr, l*elementSize))[0:elementSize:elementSize]
	if bytes.Equal(src, buf) {
		return
	}
	copy(src, buf)
}

func (m *Matrix) Inc(i, j uint64) {
	l := location(i, j)
	buf := (*[elementSize]byte)(unsafe.Add(m.dataPtr, l*elementSize))[0:elementSize:elementSize]
	incrementLittleEndian(buf)
}

func (m *Matrix) Dec(i, j uint64) {
	l := location(i, j)
	buf := (*[elementSize]byte)(unsafe.Add(m.dataPtr, l*elementSize))[0:elementSize:elementSize]
	decrementLittleEndian(buf)
}

func (m *Matrix) Sync() error {
	if err := m.mmap.sync(); err != nil {
		return fmt.Errorf("sync mmap size %d: %w", len(*m.mmap), err)
	}
	if err := m.file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	return nil
}

func (m *Matrix) Size() uint64 {
	length := uint64(len(*m.mmap))
	if length < versionSize {
		return 0
	}
	return floorSqrt((length - versionSize) / elementSize)
}

func (m *Matrix) Close() error {
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

func (m *Matrix) decode(data []byte) uint32 {
	return binary.LittleEndian.Uint32(data)
}

func (m *Matrix) encode(v uint32) []byte {
	binary.LittleEndian.PutUint32(m.encodeBuf, v)
	return m.encodeBuf
}

// matrix i,j = 0,0 0,1 1,0 1,1 0,2 1,2 2,0 2,1 2,2 0,3 1,3 2,3 3,0 3,1 3,2 3,3
// array  n   =   0   1   2   3   4   5   6   7   8   9  10  11  12  13  14  15
func location(i, j uint64) uint64 {
	// if i < j {
	// 	return j*j + i
	// }
	// return i*i + i + j
	return (i*i + i + j) - ((i-j)>>63)*(i*i+i+j-(j*j+i))
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
