// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	cluster "github.com/upbound/provider-coreweave/internal/controller/cluster/cks/cluster"
	accessconfiguration "github.com/upbound/provider-coreweave/internal/controller/cluster/containerregistry/accessconfiguration"
	lifecyclepolicy "github.com/upbound/provider-coreweave/internal/controller/cluster/containerregistry/lifecyclepolicy"
	namespace "github.com/upbound/provider-coreweave/internal/controller/cluster/containerregistry/namespace"
	capacityclaim "github.com/upbound/provider-coreweave/internal/controller/cluster/inference/capacityclaim"
	deployment "github.com/upbound/provider-coreweave/internal/controller/cluster/inference/deployment"
	gateway "github.com/upbound/provider-coreweave/internal/controller/cluster/inference/gateway"
	vpc "github.com/upbound/provider-coreweave/internal/controller/cluster/networking/vpc"
	accesskey "github.com/upbound/provider-coreweave/internal/controller/cluster/objectstorage/accesskey"
	bucket "github.com/upbound/provider-coreweave/internal/controller/cluster/objectstorage/bucket"
	bucketinventory "github.com/upbound/provider-coreweave/internal/controller/cluster/objectstorage/bucketinventory"
	bucketlifecycleconfiguration "github.com/upbound/provider-coreweave/internal/controller/cluster/objectstorage/bucketlifecycleconfiguration"
	bucketpolicy "github.com/upbound/provider-coreweave/internal/controller/cluster/objectstorage/bucketpolicy"
	bucketsettings "github.com/upbound/provider-coreweave/internal/controller/cluster/objectstorage/bucketsettings"
	bucketversioning "github.com/upbound/provider-coreweave/internal/controller/cluster/objectstorage/bucketversioning"
	organizationaccesspolicy "github.com/upbound/provider-coreweave/internal/controller/cluster/objectstorage/organizationaccesspolicy"
	providerconfig "github.com/upbound/provider-coreweave/internal/controller/cluster/providerconfig"
	managedrunner "github.com/upbound/provider-coreweave/internal/controller/cluster/sandbox/managedrunner"
	oidcconfig "github.com/upbound/provider-coreweave/internal/controller/cluster/workloadfederation/oidcconfig"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		cluster.Setup,
		accessconfiguration.Setup,
		lifecyclepolicy.Setup,
		namespace.Setup,
		capacityclaim.Setup,
		deployment.Setup,
		gateway.Setup,
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
		managedrunner.Setup,
		oidcconfig.Setup,
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
		accessconfiguration.SetupGated,
		lifecyclepolicy.SetupGated,
		namespace.SetupGated,
		capacityclaim.SetupGated,
		deployment.SetupGated,
		gateway.SetupGated,
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
		managedrunner.SetupGated,
		oidcconfig.SetupGated,
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
		accessconfiguration.SetupWebhookWithManager,
		lifecyclepolicy.SetupWebhookWithManager,
		namespace.SetupWebhookWithManager,
		capacityclaim.SetupWebhookWithManager,
		deployment.SetupWebhookWithManager,
		gateway.SetupWebhookWithManager,
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
		managedrunner.SetupWebhookWithManager,
		oidcconfig.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
