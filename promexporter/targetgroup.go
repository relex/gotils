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

	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

// Target is a prometheus scrape target.
//
// Labels must be a comparable struct, see `GetLabelNames` function description.
type Target[L comparable] struct {
	Target string // address, e.g. "host.org:12340"
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
	targetGroupMap := lo.GroupBy(targetList, func(t Target[L]) L {
		return t.Labels
	})

	targetGroups := lo.MapToSlice(targetGroupMap, func(labels L, targets []Target[L]) TargetGroup[L] {
		targetAddresses := lo.Map(targets, func(target Target[L], _ int) string {
			return target.Target
		})
		slices.Sort(targetAddresses)

		return TargetGroup[L]{
			Targets: targetAddresses,
			Labels:  labels,
		}
	})

	slices.SortStableFunc(targetGroups, func(a, b TargetGroup[L]) bool {
		return fmt.Sprint(a.Labels) < fmt.Sprint(b.Labels)
	})

	return targetGroups
}
