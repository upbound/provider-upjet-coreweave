// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package config

import (
	"strings"

	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/types/name"
)

// GroupKindOverrides overrides the group and kind of the resource if it matches
// any entry in the GroupMap.
func GroupKindOverrides() config.ResourceOption {
	return func(r *config.Resource) {
		if f, ok := GroupMap[r.Name]; ok {
			r.ShortGroup, r.Kind = f(r.Name)
		}
	}
}

// GroupKindCalculator returns the correct group and kind name for given TF
// resource.
type GroupKindCalculator func(resource string) (string, string)

// ReplaceGroupWords uses given group as the group of the resource and removes
// a number of words in resource name before calculating the kind of the resource.
func ReplaceGroupWords(group string, count int) GroupKindCalculator {
	return func(resource string) (string, string) {
		// "coreweave_vpc_subnet" with ("vpc", 1) -> (vpc, Subnet)
		words := strings.Split(strings.TrimPrefix(resource, resourcePrefix+"_"), "_")
		snakeKind := strings.Join(words[count:], "_")
		return group, name.NewFromSnake(snakeKind).Camel
	}
}

// KnownGroupKind returns a GroupKindCalculator that assigns the given static
// group and kind regardless of the resource name.
func KnownGroupKind(group, kind string) GroupKindCalculator {
	return func(string) (string, string) { return group, kind }
}

// GroupMap contains all overrides we'd like to make to the default group search.
// Without an entry, upjet derives the group from the first word after the
// resource prefix and the kind from the rest.
var GroupMap = map[string]GroupKindCalculator{
	"coreweave_networking_vpc":                                KnownGroupKind("networking", "VPC"),
	"coreweave_cks_cluster":                                   ReplaceGroupWords("cks", 1),
	"coreweave_object_storage_bucket":                         ReplaceGroupWords("objectstorage", 2),
	"coreweave_object_storage_bucket_inventory":               ReplaceGroupWords("objectstorage", 2),
	"coreweave_object_storage_bucket_lifecycle_configuration": ReplaceGroupWords("objectstorage", 2),
	"coreweave_object_storage_bucket_policy":                  ReplaceGroupWords("objectstorage", 2),
	"coreweave_object_storage_bucket_settings":                ReplaceGroupWords("objectstorage", 2),
	"coreweave_object_storage_bucket_versioning":              ReplaceGroupWords("objectstorage", 2),
	"coreweave_object_storage_access_key":                     ReplaceGroupWords("objectstorage", 2),
	"coreweave_object_storage_organization_access_policy":     ReplaceGroupWords("objectstorage", 2),
	"coreweave_inference_capacity_claim":                      ReplaceGroupWords("inference", 1),
	"coreweave_inference_deployment":                          ReplaceGroupWords("inference", 1),
	"coreweave_inference_gateway":                             ReplaceGroupWords("inference", 1),
	"coreweave_workload_federation_oidc_config":               KnownGroupKind("workloadfederation", "OIDCConfig"),
	"coreweave_container_registry_namespace":                  ReplaceGroupWords("containerregistry", 2),
	"coreweave_container_registry_access_configuration":       ReplaceGroupWords("containerregistry", 2),
	"coreweave_container_registry_lifecycle_policy":           ReplaceGroupWords("containerregistry", 2),
}
