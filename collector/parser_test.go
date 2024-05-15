/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ExtractVersions(t *testing.T) {
	tests := []struct {
		name             string
		version          string
		less             string
		wantIntroduce    string
		wantLastAffected string
	}{
		{name: "range less with minor", version: "1.2", less: "1.2.5", wantIntroduce: "1.2.0", wantLastAffected: "1.2.5"},
		{name: "range less", version: "", less: "1.2.5", wantIntroduce: "1.2.0", wantLastAffected: "1.2.5"},
		{name: "range lessThen", version: "", less: "1.2.5", wantIntroduce: "1.2.0", wantLastAffected: "1.2.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIntoduce, gotLastAffected := updateVersions(tt.less, tt.version)
			assert.Equal(t, gotIntoduce, tt.wantIntroduce)
			assert.Equal(t, gotLastAffected, tt.wantLastAffected)
		})
	}
}

func Test_ExtractRangeVersions(t *testing.T) {
	tests := []struct {
		name             string
		version          string
		wantIntroduce    string
		wantLastAffected string
	}{
		{name: "range versions", version: "1.2.3 - 1.2.5", wantIntroduce: "1.2.3", wantLastAffected: "1.2.5"},
		{name: "single versions", version: "1.2.5", wantIntroduce: "1.2.5", wantLastAffected: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIntoduce, gotLastAffected := extractRangeVersions(tt.version)
			assert.Equal(t, gotIntoduce, tt.wantIntroduce)
			assert.Equal(t, gotLastAffected, tt.wantLastAffected)
		})
	}
}
