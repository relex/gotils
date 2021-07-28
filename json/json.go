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

package json

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// MarshalJSONWithSorting marshals JSON with keys sorted by alphabet, same rule as marshalling map
//
// When called from a Marshaller, the input should be a new/derived type without Marshaller defined to prevent infinite recursion
func MarshalJSONWithSorting(input interface{}) ([]byte, error) {
	mJSON, mErr := json.Marshal(input)
	if mErr != nil {
		return nil, fmt.Errorf("error marshalling intermediate input: %v: %w", input, mErr)
	}
	// remarshal from map to sort labels
	fieldMap := make(map[string]interface{})
	if err := json.Unmarshal(mJSON, &fieldMap); err != nil {
		return nil, fmt.Errorf("error unmarshalling intermediate field map: %v: %w", fieldMap, err)
	}
	sortedJSON, sErr := json.Marshal(fieldMap)
	if sErr != nil {
		return nil, fmt.Errorf("error marshalling intermediate field map: %v: %w", fieldMap, sErr)
	}
	return sortedJSON, nil
}

// MarshalToJSONFile marshals structure to a JSON file at the specified path
func MarshalToJSONFile(filepath string, input interface{}) error {
	data, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath, data, 0644)
}

// UnmarshalFromJSONFile unmarshals JSON file at the specified path
func UnmarshalFromJSONFile(filepath string, outputPtr interface{}) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, outputPtr)
}
