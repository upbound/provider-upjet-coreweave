// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package cluster

import (
	"github.com/upbound/provider-coreweave/config/cluster/cks"
	"github.com/upbound/provider-coreweave/config/cluster/inference"
	"github.com/upbound/provider-coreweave/config/cluster/objectstorage"
)

// Keep this list identical to config/namespaced/provider.go.
func init() {
	ProviderConfiguration.AddConfig(cks.Configure)
	ProviderConfiguration.AddConfig(objectstorage.Configure)
	ProviderConfiguration.AddConfig(inference.Configure)
}
