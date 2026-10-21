// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package config

import (
	"context"

	"github.com/coreweave/terraform-provider-coreweave/xpprovider"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The upstream Terraform provider is linked into this binary and driven
// in-process by upjet: there is no terraform CLI at runtime. The CoreWeave
// provider is written against terraform-plugin-framework only, so the SDKv2
// constructor stays nil.
//
// Both functions are called once per scope (cluster and namespaced), so each
// call must return a fresh provider instance.

// upstreamProviderVersion is the version the upstream provider reports in the
// User-Agent of every CoreWeave API call. Keep it in sync with
// TERRAFORM_PROVIDER_VERSION in the Makefile and the module version in go.mod.
const upstreamProviderVersion = "0.26.0"

// sdkProvider returns the upstream terraform-plugin-sdk/v2 provider. The
// CoreWeave provider implements no SDKv2 resources.
func sdkProvider() *schema.Provider { //nolint:unparam // the upstream provider has no SDKv2 half
	return nil
}

// frameworkProvider returns a fresh instance of the upstream
// terraform-plugin-framework provider through the xpprovider package of the
// upjet branch of the CoreWeave provider.
func frameworkProvider(ctx context.Context) (fwprovider.Provider, error) {
	return xpprovider.GetProvider(ctx, upstreamProviderVersion)
}
