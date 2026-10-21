// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package sandbox

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

const extractID = `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("id",true)`

// Configure configures the sandbox group.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("coreweave_sandbox_managed_runner", func(r *config.Resource) {
		r.References["cluster_id"] = config.Reference{
			TerraformName: "coreweave_cks_cluster",
			Extractor:     extractID,
		}
	})
}
