// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus

func (v *Voting) StrengthsMatrixFilename() string {
	return v.strengthsMatrixFilename()
}

func (v *Voting) InvalidateStrengthMatrix() error {
	return v.strengths.SetVersion(0)
}
