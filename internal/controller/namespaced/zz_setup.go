// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	cluster "github.com/upbound/provider-coreweave/internal/controller/namespaced/cks/cluster"
	vpc "github.com/upbound/provider-coreweave/internal/controller/namespaced/networking/vpc"
	accesskey "github.com/upbound/provider-coreweave/internal/controller/namespaced/objectstorage/accesskey"
	bucket "github.com/upbound/provider-coreweave/internal/controller/namespaced/objectstorage/bucket"
	bucketinventory "github.com/upbound/provider-coreweave/internal/controller/namespaced/objectstorage/bucketinventory"
	bucketlifecycleconfiguration "github.com/upbound/provider-coreweave/internal/controller/namespaced/objectstorage/bucketlifecycleconfiguration"
	bucketpolicy "github.com/upbound/provider-coreweave/internal/controller/namespaced/objectstorage/bucketpolicy"
	bucketsettings "github.com/upbound/provider-coreweave/internal/controller/namespaced/objectstorage/bucketsettings"
	bucketversioning "github.com/upbound/provider-coreweave/internal/controller/namespaced/objectstorage/bucketversioning"
	organizationaccesspolicy "github.com/upbound/provider-coreweave/internal/controller/namespaced/objectstorage/organizationaccesspolicy"
	providerconfig "github.com/upbound/provider-coreweave/internal/controller/namespaced/providerconfig"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		cluster.Setup,
		vpc.Setup,
		accesskey.Setup,
		bucket.Setup,
		bucketinventory.Setup,
		bucketlifecycleconfiguration.Setup,
		bucketpolicy.Setup,
		bucketsettings.Setup,
		bucketversioning.Setup,
		organizationaccesspolicy.Setup,
		providerconfig.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		cluster.SetupGated,
		vpc.SetupGated,
		accesskey.SetupGated,
		bucket.SetupGated,
		bucketinventory.SetupGated,
		bucketlifecycleconfiguration.SetupGated,
		bucketpolicy.SetupGated,
		bucketsettings.SetupGated,
		bucketversioning.SetupGated,
		organizationaccesspolicy.SetupGated,
		providerconfig.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhookWithManager registers conversion webhooks for all resource kinds in the group.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		cluster.SetupWebhookWithManager,
		vpc.SetupWebhookWithManager,
		accesskey.SetupWebhookWithManager,
		bucket.SetupWebhookWithManager,
		bucketinventory.SetupWebhookWithManager,
		bucketlifecycleconfiguration.SetupWebhookWithManager,
		bucketpolicy.SetupWebhookWithManager,
		bucketsettings.SetupWebhookWithManager,
		bucketversioning.SetupWebhookWithManager,
		organizationaccesspolicy.SetupWebhookWithManager,
		providerconfig.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
