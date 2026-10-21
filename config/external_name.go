// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package config

import (
	"sort"

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
