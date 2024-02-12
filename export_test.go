// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus

import "fmt"

func (v *Voting) StrengthsMatrixFilename() string {
	return v.strengthsMatrixFilename()
}

func (v *Voting) InvalidateStrengthMatrix() error {
	return v.strengths.SetVersion(0)
}

func (v *Voting) PreferencesMatrix() []uint32 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	preferences := make([]uint32, 0)

	for i := uint64(0); i < v.preferencesSize; i++ {
		for j := uint64(0); j < v.preferencesSize; j++ {
			preferences = append(preferences, v.preferences.Get(i, j))
		}
	}

	return preferences
}

func (v *Voting) PreferencesMatrixLabels() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	choices := make([]string, 0)

	for i := uint64(0); i < v.preferences.Size(); i++ {
		virtual, has := v.choicesIndex.GetVirtual(i)
		vs := fmt.Sprint(virtual)
		if !has {
			vs = "/"
		}
		choices = append(choices, fmt.Sprintf("%v (%s)", i, vs))
	}

	return choices
}

func (v *Voting) Choices() []uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	choices := make([]uint64, 0)

	for i := uint64(0); i < v.preferences.Size(); i++ {
		i, has := v.choicesIndex.GetVirtual(i)
		if !has {
			continue
		}
		choices = append(choices, i)
	}

	return choices
}
