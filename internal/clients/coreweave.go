package clients

import (
	"context"
	"encoding/json"
	"sync"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upjetresource "github.com/crossplane/upjet/v2/pkg/resource"
	"github.com/crossplane/upjet/v2/pkg/terraform"

	clusterv1beta1 "github.com/upbound/provider-coreweave/apis/cluster/v1beta1"
	namespacedv1beta1 "github.com/upbound/provider-coreweave/apis/namespaced/v1beta1"
)

const (
	// error messages
	errResolveProviderConfig = "cannot resolve provider config"
	errNoProviderConfig      = "no providerConfigRef provided"
	errGetProviderConfig     = "cannot get referenced ProviderConfig"
	errTrackUsage            = "cannot track ProviderConfig usage"
	errExtractCredentials    = "cannot extract credentials"
	errUnmarshalCredentials  = "cannot unmarshal coreweave credentials as JSON"
	errBuildConfiguration    = "cannot build the CoreWeave provider configuration"
	errConfigureSDKProvider  = "cannot configure the CoreWeave SDKv2 provider"

	// arguments of the upstream provider configuration block
	keyToken       = "token"
	keyEndpoint    = "endpoint"
	keyS3Endpoint  = "s3_endpoint"
	keyHTTPTimeout = "http_timeout"
)

// sdkConfigureMu serializes configuration of the shared SDKv2 provider, whose
// Configure stores the resulting client in provider-wide state.
var sdkConfigureMu sync.Mutex

// TerraformSetupBuilder builds a terraform.SetupFn that configures the
// in-process upstream Terraform providers (no terraform CLI). The plugin
// framework provider is configured by upjet's connector from the returned
// Configuration. The SDKv2 provider is configured here, for SDKv2 resources
// only, because its connector consumes the configured client from Setup.Meta.
func TerraformSetupBuilder(sdkProvider *schema.Provider, fwProvider fwprovider.Provider) terraform.SetupFn {
	return func(ctx context.Context, client client.Client, mg resource.Managed) (terraform.Setup, error) {
		pcSpec, err := resolveProviderConfig(ctx, client, mg)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, errResolveProviderConfig)
		}

		data, err := resource.CommonCredentialExtractor(ctx, pcSpec.Credentials.Source, client, pcSpec.Credentials.CommonCredentialSelectors)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, errExtractCredentials)
		}

		creds := map[string]any{}
		if err := json.Unmarshal(data, &creds); err != nil {
			return terraform.Setup{}, errors.Wrap(err, errUnmarshalCredentials)
		}

		cfg, err := buildConfiguration(creds, pcSpec)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, errBuildConfiguration)
		}

		setup := terraform.Setup{
			Configuration:     cfg,
			FrameworkProvider: fwProvider,
		}
		if isSDKv2Resource(sdkProvider, mg) {
			meta, err := configureSDKProvider(ctx, sdkProvider, setup.Configuration)
			if err != nil {
				return terraform.Setup{}, err
			}
			setup.Meta = meta
		}
		return setup, nil
	}
}

// buildConfiguration maps the credentials JSON and the ProviderConfig spec
// onto the CoreWeave provider configuration block. The API token comes from
// the credentials secret and is required; the endpoint, S3 endpoint and HTTP
// timeout come from the spec and are passed through when set.
func buildConfiguration(creds map[string]any, spec *namespacedv1beta1.ProviderConfigSpec) (map[string]any, error) {
	token, _ := creds[keyToken].(string)
	if token == "" {
		return nil, errors.Errorf("credentials secret has no %q key", keyToken)
	}
	cfg := map[string]any{keyToken: token}
	for key, value := range map[string]*string{
		keyEndpoint:    spec.Endpoint,
		keyS3Endpoint:  spec.S3Endpoint,
		keyHTTPTimeout: spec.HTTPTimeout,
	} {
		if value != nil && *value != "" {
			cfg[key] = *value
		}
	}
	return cfg, nil
}

// isSDKv2Resource reports whether the managed resource is implemented by the
// SDKv2 provider, which is the only one that needs to be configured here.
func isSDKv2Resource(p *schema.Provider, mg resource.Managed) bool {
	tr, ok := mg.(upjetresource.Terraformed)
	if !ok || p == nil {
		return false
	}
	_, isSDK := p.ResourcesMap[tr.GetTerraformResourceType()]
	return isSDK
}

func configureSDKProvider(ctx context.Context, p *schema.Provider, cfg map[string]any) (any, error) {
	sdkConfigureMu.Lock()
	defer sdkConfigureMu.Unlock()
	if diags := p.Configure(ctx, sdkterraform.NewResourceConfigRaw(cfg)); diags.HasError() {
		return nil, errors.Errorf("%s: %v", errConfigureSDKProvider, diags)
	}
	return p.Meta(), nil
}

func toSharedPCSpec(pc *clusterv1beta1.ProviderConfig) (*namespacedv1beta1.ProviderConfigSpec, error) {
	if pc == nil {
		return nil, nil
	}
	data, err := json.Marshal(pc.Spec)
	if err != nil {
		return nil, err
	}

	var mSpec namespacedv1beta1.ProviderConfigSpec
	err = json.Unmarshal(data, &mSpec)
	return &mSpec, err
}

func resolveProviderConfig(ctx context.Context, crClient client.Client, mg resource.Managed) (*namespacedv1beta1.ProviderConfigSpec, error) {
	switch managed := mg.(type) {
	case resource.LegacyManaged: //nolint:staticcheck // still handling cluster-scoped behavior
		return resolveLegacy(ctx, crClient, managed)
	case resource.ModernManaged:
		return resolveModern(ctx, crClient, managed)
	default:
		return nil, errors.New("resource is not a managed resource")
	}
}

func resolveLegacy(ctx context.Context, client client.Client, mg resource.LegacyManaged) (*namespacedv1beta1.ProviderConfigSpec, error) { //nolint:staticcheck // still handling cluster-scoped behavior
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}
	pc := &clusterv1beta1.ProviderConfig{}
	if err := client.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	t := resource.NewLegacyProviderConfigUsageTracker(client, &clusterv1beta1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}

	return toSharedPCSpec(pc)
}

// resolveNamespacedSpec converts a namespaced ProviderConfig spec into the
// internal, fully-resolved ProviderConfigSpec. The namespaced spec omits the
// secret namespace, so it is resolved to the namespace of the referencing
// managed resource here.
func resolveNamespacedSpec(spec namespacedv1beta1.NamespacedProviderConfigSpec, namespace string) namespacedv1beta1.ProviderConfigSpec {
	resolved := namespacedv1beta1.ProviderConfigSpec{
		ReconciliationPolicy: spec.ReconciliationPolicy,
		Endpoint:             spec.Endpoint,
		S3Endpoint:           spec.S3Endpoint,
		HTTPTimeout:          spec.HTTPTimeout,
		Credentials: namespacedv1beta1.ProviderCredentials{
			Source: spec.Credentials.Source,
			CommonCredentialSelectors: xpv2.CommonCredentialSelectors{
				Fs:  spec.Credentials.Fs,
				Env: spec.Credentials.Env,
			},
		},
	}
	if spec.Credentials.SecretRef != nil {
		resolved.Credentials.SecretRef = spec.Credentials.SecretRef.ToSecretKeySelector(namespace)
	}
	return resolved
}

func resolveModern(ctx context.Context, crClient client.Client, mg resource.ModernManaged) (*namespacedv1beta1.ProviderConfigSpec, error) {
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}

	pcRuntimeObj, err := crClient.Scheme().New(namespacedv1beta1.SchemeGroupVersion.WithKind(configRef.Kind))
	if err != nil {
		return nil, errors.Wrap(err, "unknown GVK for ProviderConfig")
	}
	pcObj, ok := pcRuntimeObj.(client.Object)
	if !ok {
		// This indicates a programming error, types are not properly generated
		return nil, errors.New("runtime object is not a client.Object")
	}

	// A ClusterProviderConfig is cluster-scoped. The API server ignores a
	// namespace on its lookup, but the diff server reads it from an in-memory
	// store keyed on the exact namespace, so it must be cleared here rather
	// than left to the client.
	ns := mg.GetNamespace()
	if configRef.Kind == namespacedv1beta1.ClusterProviderConfigKind {
		ns = ""
	}
	if err := crClient.Get(ctx, types.NamespacedName{Name: configRef.Name, Namespace: ns}, pcObj); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	var pcSpec namespacedv1beta1.ProviderConfigSpec
	pcu := &namespacedv1beta1.ProviderConfigUsage{}
	switch pc := pcObj.(type) {
	case *namespacedv1beta1.ProviderConfig:
		pcSpec = resolveNamespacedSpec(pc.Spec, mg.GetNamespace())
	case *namespacedv1beta1.ClusterProviderConfig:
		pcSpec = pc.Spec
	default:
		return nil, errors.New("unknown provider config type")
	}
	t := resource.NewProviderConfigUsageTracker(crClient, pcu)
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}
	return &pcSpec, nil
}
