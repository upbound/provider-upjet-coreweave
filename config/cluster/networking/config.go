// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package networking

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the networking group.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("coreweave_networking_vpc", func(r *config.Resource) {
		// The whole blocks are ignored: ignoring only the nested booleans still
		// late-initializes an empty object. Remove once
		// https://github.com/coreweave/terraform-provider-coreweave/pull/554 is merged.
		r.LateInitializer = config.LateInitializer{
			IgnoredFields: []string{"egress", "ingress"},
		}
	})
}
