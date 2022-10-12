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
	"fmt"

	"github.com/relex/gotils/generics"
	"golang.org/x/exp/slices"
)

// Target is a prometheus scrape target.
//
// Labels must be a comparable struct, see `GetLabelNames` function description.
type Target[L comparable] struct {
	Target string
	Labels L
}

// TargetGroup is a group of scrape targets sharing the same labels.
//
// Each group normally represent the deployment one cluster on one server.
type TargetGroup[L comparable] struct {
	Targets []string `json:"targets"`
	Labels  L        `json:"labels"`
}

// GroupTargets groups targets by distinct label values
//
// The resulting target groups and the target addresses inside each group are ordered by their string representations
// to ensure stable ordering.
func GroupTargets[L comparable](targetList []Target[L]) []TargetGroup[L] {
	groupMap := generics.GroupSlice(targetList, func(t Target[L]) L {
		return t.Labels
	})

	return generics.MapToSliceWithSortFunc(
		groupMap,

		func(labels L, targets []Target[L]) TargetGroup[L] {
			targetAddresses := generics.MapSlice(targets, func(target Target[L]) string {
				return target.Target
			})
			slices.Sort(targetAddresses)

			return TargetGroup[L]{
				Targets: targetAddresses,
				Labels:  labels,
			}
		},

		func(labels1, labels2 L) bool {
			return fmt.Sprint(labels1) < fmt.Sprint(labels2)
		},
	)
}
