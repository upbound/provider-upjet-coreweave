// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package namespaced

import (
	"github.com/upbound/provider-coreweave/config/namespaced/cks"
	"github.com/upbound/provider-coreweave/config/namespaced/containerregistry"
	"github.com/upbound/provider-coreweave/config/namespaced/inference"
	"github.com/upbound/provider-coreweave/config/namespaced/objectstorage"
)

// Keep this list identical to config/cluster/provider.go.
func init() {
	ProviderConfiguration.AddConfig(cks.Configure)
	ProviderConfiguration.AddConfig(objectstorage.Configure)
	ProviderConfiguration.AddConfig(inference.Configure)
	ProviderConfiguration.AddConfig(containerregistry.Configure)
}
