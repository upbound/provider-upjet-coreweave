// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package containerregistry

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the containerregistry group. The per-namespace
// singletons reference the Namespace by its external name, which is the
// namespace name.
func Configure(p *config.Provider) {
	for _, name := range []string{
		"coreweave_container_registry_access_configuration",
		"coreweave_container_registry_lifecycle_policy",
	} {
		p.AddResourceConfigurator(name, func(r *config.Resource) {
			r.References["namespace"] = config.Reference{
				TerraformName: "coreweave_container_registry_namespace",
			}
		})
	}
}
