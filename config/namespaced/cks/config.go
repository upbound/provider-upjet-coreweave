// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package cks

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

const extractID = `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("id",true)`

// Configure configures the cks group.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("coreweave_cks_cluster", func(r *config.Resource) {
		r.References["vpc_id"] = config.Reference{
			TerraformName: "coreweave_networking_vpc",
			Extractor:     extractID,
		}
		r.References["shared_storage_cluster_id"] = config.Reference{
			TerraformName: "coreweave_cks_cluster",
			Extractor:     extractID,
		}
	})
}
