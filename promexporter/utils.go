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

// Target is a prometheus scrape target
// Labels must be a comparable struct, see `GetLabelNames` function description
type Target struct {
	Target string
	Labels interface{}
}

// TargetGroup is a group of scrape targets sharing the same labels.
// Each group normally represent the deployment one cluster on one server.
type TargetGroup struct {
	Targets []string    `json:"targets"`
	Labels  interface{} `json:"labels"`
}

// GroupTargets groups targets by distinct label values
func GroupTargets(targetList []Target) []TargetGroup {
	targetGroupByLabels := make(map[interface{}]TargetGroup)

	for _, target := range targetList {
		if group, exists := targetGroupByLabels[target.Labels]; exists {
			group.Targets = append(group.Targets, target.Target)
		} else {
			targetGroupByLabels[target.Labels] = TargetGroup{
				Targets: []string{target.Target},
				Labels:  target.Labels,
			}
		}
	}

	result := make([]TargetGroup, len(targetGroupByLabels))
	i := 0
	for _, group := range targetGroupByLabels {
		result[i] = group
		i++
	}
	return result
}
