// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package config

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"

	"github.com/upbound/provider-coreweave/config/namespaced"
	"github.com/upbound/provider-coreweave/config/templates"
)

// GetProviderNamespaced returns the namespaced provider configuration.
func GetProviderNamespaced(ctx context.Context) (*ujconfig.Provider, error) {
	fwProvider, err := frameworkProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errUpstreamProvider)
	}

	pc := ujconfig.NewProvider(
		[]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup("coreweave.m.upbound.io"),
		ujconfig.WithIncludeList([]string{}),
		ujconfig.WithControllerTemplate(templates.ControllerTemplate),
		ujconfig.WithTerraformProvider(sdkProvider()),
		ujconfig.WithTerraformPluginSDKIncludeList(terraformPluginSDKResourceList()),
		ujconfig.WithTerraformPluginFrameworkProvider(fwProvider),
		ujconfig.WithTerraformPluginFrameworkIncludeList(terraformPluginFrameworkResourceList()),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithSchemaTraversers(&ujconfig.SingletonListEmbedder{}),
		ujconfig.WithDefaultResourceOptions(defaultResourceOptions()...),
	)

	// add custom config functions
	for _, configure := range namespaced.ProviderConfiguration {
		configure(pc)
	}

	pc.ConfigureResources()

	registerTFConversions(pc)

	return pc, nil
}
