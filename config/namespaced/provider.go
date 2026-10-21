// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package namespaced

import (
	"github.com/upbound/provider-coreweave/config/namespaced/cks"
)

// Keep this list identical to config/cluster/provider.go.
func init() {
	ProviderConfiguration.AddConfig(cks.Configure)
}
