// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package config

import (
	"sort"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/upjet/v2/pkg/config"
)

// nilUUID stands in for a server-assigned UUID identifier until the resource
// exists; the CoreWeave API answers not-found for it.
const nilUUID = "00000000-0000-0000-0000-000000000000"

// terraformPluginSDKExternalNameConfigs contains the external name
// configuration of every resource the upstream provider implements with
// terraform-plugin-sdk/v2, keyed by Terraform resource name. Listing a
// resource here is what makes the generator emit it, with the in-process SDK
// client; its provider meta is configured in internal/clients.
var terraformPluginSDKExternalNameConfigs = map[string]config.ExternalName{
	// "coreweave_instance": config.IdentifierFromProvider,
}

// terraformPluginFrameworkExternalNameConfigs is the same registry for the
// resources the upstream provider implements with terraform-plugin-framework.
// A resource must appear in at most one of the two maps.
var terraformPluginFrameworkExternalNameConfigs = map[string]config.ExternalName{
	// VPC ids are server-assigned UUIDs.
	"coreweave_networking_vpc": config.FrameworkResourceWithComputedIdentifier("id", nilUUID),
	"coreweave_cks_cluster":    config.FrameworkResourceWithComputedIdentifier("id", nilUUID),
	// Object storage buckets and organization access policies have no id
	// attribute; the name is the identity. The bucket configuration resources
	// are identified by the bucket they configure (plus the inventory name);
	// the external name is assigned after creation so that bucket keeps its
	// reference. Access key ids are assigned by the API.
	"coreweave_object_storage_bucket":                         frameworkNameAsIdentifier(),
	"coreweave_object_storage_bucket_inventory":               externalNameFromArguments("bucket", "name"),
	"coreweave_object_storage_bucket_lifecycle_configuration": externalNameFromArguments("bucket"),
	"coreweave_object_storage_bucket_policy":                  externalNameFromArguments("bucket"),
	"coreweave_object_storage_bucket_settings":                externalNameFromArguments("bucket"),
	"coreweave_object_storage_bucket_versioning":              externalNameFromArguments("bucket"),
	"coreweave_object_storage_access_key":                     config.FrameworkResourceWithComputedIdentifier("id", nilUUID),
	"coreweave_object_storage_organization_access_policy":     frameworkNameAsIdentifier(),
	"coreweave_inference_capacity_claim":                      config.FrameworkResourceWithComputedIdentifier("id", nilUUID),
	"coreweave_inference_deployment":                          config.FrameworkResourceWithComputedIdentifier("id", nilUUID),
	"coreweave_inference_gateway":                             config.FrameworkResourceWithComputedIdentifier("id", nilUUID),
	"coreweave_workload_federation_oidc_config":               config.FrameworkResourceWithComputedIdentifier("id", nilUUID),
}

// frameworkNameAsIdentifier identifies a framework resource that has no id
// attribute by its name argument: the external name drives name, which is
// dropped from the spec. The upjet default reads the external name back from
// id, which such a resource never has in its state.
func frameworkNameAsIdentifier() config.ExternalName {
	e := config.NameAsIdentifier
	e.GetExternalNameFn = externalNameFromState("name")
	return e
}

// externalNameFromArguments identifies a framework resource that has no id
// attribute by the given arguments, joined with ":" in the import format of
// the upstream provider. The arguments stay in the spec, so they can carry
// references, and the external name is read back from the state after
// creation, as for config.IdentifierFromProvider.
func externalNameFromArguments(arguments ...string) config.ExternalName {
	return config.NewExternalNameFrom(config.IdentifierFromProvider,
		config.WithGetExternalNameFn(func(_ config.GetExternalNameFn, tfState map[string]any) (string, error) {
			return externalNameFromState(arguments...)(tfState)
		}),
	)
}

func externalNameFromState(arguments ...string) config.GetExternalNameFn {
	return func(tfState map[string]any) (string, error) {
		parts := make([]string, 0, len(arguments))
		for _, argument := range arguments {
			v, ok := tfState[argument].(string)
			if !ok || v == "" {
				return "", errors.Errorf("cannot find attribute %q in tfstate", argument)
			}
			parts = append(parts, v)
		}
		return strings.Join(parts, ":"), nil
	}
}

// ExternalNameConfigurations applies all external name configs listed in the
// tables terraformPluginSDKExternalNameConfigs and
// terraformPluginFrameworkExternalNameConfigs and sets the version of those
// resources to v1beta1 assuming they will be tested.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		e, configured := terraformPluginFrameworkExternalNameConfigs[r.Name]
		if !configured {
			e, configured = terraformPluginSDKExternalNameConfigs[r.Name]
		}
		if !configured {
			return
		}
		r.ExternalName = e
		r.Version = versionV1Beta1
	}
}

// terraformPluginSDKResourceList returns the list of resources generated with
// the terraform-plugin-sdk/v2 client, anchored for the include-list regex.
func terraformPluginSDKResourceList() []string {
	return resourceList(terraformPluginSDKExternalNameConfigs)
}

// terraformPluginFrameworkResourceList returns the list of resources
// generated with the terraform-plugin-framework client, anchored for the
// include-list regex.
func terraformPluginFrameworkResourceList() []string {
	return resourceList(terraformPluginFrameworkExternalNameConfigs)
}

func resourceList(t map[string]config.ExternalName) []string {
	l := make([]string, 0, len(t))
	for name := range t {
		// $ is added to match the exact string since the format is regex.
		l = append(l, name+"$")
	}
	sort.Strings(l)
	return l
}
