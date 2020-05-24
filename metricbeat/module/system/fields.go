// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Code generated by beats/dev-tools/cmd/asset/asset.go - DO NOT EDIT.

package system

import (
	"github.com/elastic/beats/libbeat/asset"
)

func init() {
	if err := asset.SetFields("metricbeat", "system", asset.ModuleFieldsPri, AssetSystem); err != nil {
		panic(err)
	}
}

// AssetSystem returns asset data.
// This is the base64 encoded gzipped contents of ../metricbeat/module/system.
func AssetSystem() string {
	return "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"
}
