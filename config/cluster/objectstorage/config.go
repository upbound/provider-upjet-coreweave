// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package objectstorage

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the objectstorage group. The bucket configuration
// resources reference the Bucket by its external name, which is the bucket
// name.
func Configure(p *config.Provider) {
	for _, name := range []string{
		"coreweave_object_storage_bucket_inventory",
		"coreweave_object_storage_bucket_lifecycle_configuration",
		"coreweave_object_storage_bucket_policy",
		"coreweave_object_storage_bucket_settings",
		"coreweave_object_storage_bucket_versioning",
	} {
		p.AddResourceConfigurator(name, func(r *config.Resource) {
			r.References["bucket"] = config.Reference{
				TerraformName: "coreweave_object_storage_bucket",
			}
		})
	}
}
