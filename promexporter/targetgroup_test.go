// Copyright 2021 RELEX Oy
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package promexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type labelSet struct {
	Name  string
	Color string
}

func TestGrouping(t *testing.T) {
	assert.Equal(
		t,
		[]TargetGroup[labelSet]{
			{Targets: []string{"host1a", "host1b", "host1c"}, Labels: labelSet{"1", "red"}},
			{Targets: []string{"host2"}, Labels: labelSet{"2", "yellow"}},
		},
		GroupTargets([]Target[labelSet]{
			{
				Target: "host1c",
				Labels: labelSet{"1", "red"},
			},
			{
				Target: "host2",
				Labels: labelSet{"2", "yellow"},
			},
			{
				Target: "host1b",
				Labels: labelSet{"1", "red"},
			},
			{
				Target: "host1a",
				Labels: labelSet{"1", "red"},
			},
		}),
	)
}
