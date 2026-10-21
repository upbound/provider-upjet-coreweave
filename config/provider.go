// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package config

import (
	// Note(turkenh): we are importing this to embed provider schema document
	"context"
	_ "embed"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"

	"github.com/upbound/provider-coreweave/config/cluster"
	"github.com/upbound/provider-coreweave/config/templates"
)

const (
	resourcePrefix = "coreweave"
	modulePath     = "github.com/upbound/provider-coreweave"
	versionV1Beta1 = "v1beta1"

	errUpstreamProvider = "cannot instantiate the upstream CoreWeave Terraform provider"
)

//go:embed schema.json
var providerSchema string

//go:embed provider-metadata.yaml
var providerMetadata string

// GetProvider returns the cluster-scoped provider configuration.
func GetProvider(ctx context.Context) (*ujconfig.Provider, error) {
	fwProvider, err := frameworkProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errUpstreamProvider)
	}

	pc := ujconfig.NewProvider(
		[]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup("coreweave.upbound.io"),
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
	for _, configure := range cluster.ProviderConfiguration {
		configure(pc)
	}

	pc.ConfigureResources()

	registerTFConversions(pc)

	return pc, nil
}

// defaultResourceOptions are applied to every generated resource, for both
// scopes, before the per-group configurators run.
func defaultResourceOptions() []ujconfig.ResourceOption {
	return []ujconfig.ResourceOption{
		GroupKindOverrides(),
		ExternalNameConfigurations(),
	}
}

// registerTFConversions wires the Terraform-side conversion for every
// resource whose singleton lists were embedded as objects by the
// SingletonListEmbedder, so the controller reconciles on the converted API.
func registerTFConversions(pc *ujconfig.Provider) {
	for name, r := range pc.Resources {
		// nothing to do if no singleton list has been converted to
		// an embedded object
		if len(r.CRDListConversionPaths()) == 0 {
			continue
		}

		// the controller will be reconciling on the CRD API version
		// with the converted API (with embedded objects in place of
		// singleton lists), so we need the appropriate Terraform
		// converter in this case.
		r.TerraformConversions = []ujconfig.TerraformConversion{
			ujconfig.NewTFSingletonConversion(),
		}

		pc.Resources[name] = r
	}
}
