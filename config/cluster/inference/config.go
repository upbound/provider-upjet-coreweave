// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package inference

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the inference group.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("coreweave_inference_deployment", func(r *config.Resource) {
		r.References["gateway_ids"] = config.Reference{
			TerraformName: "coreweave_inference_gateway",
			Extractor:     `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("id",true)`,
		}
	})
}
