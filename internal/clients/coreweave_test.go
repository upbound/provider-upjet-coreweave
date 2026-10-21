package clients

import (
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	namespacedv1beta1 "github.com/upbound/provider-coreweave/apis/namespaced/v1beta1"
)

func TestResolveNamespacedSpec(t *testing.T) {
	type args struct {
		spec      namespacedv1beta1.NamespacedProviderConfigSpec
		namespace string
	}

	cases := map[string]struct {
		args args
		want namespacedv1beta1.ProviderConfigSpec
	}{
		"SecretRefResolvesToReferencerNamespace": {
			args: args{
				namespace: exampleNamespace,
				spec: namespacedv1beta1.NamespacedProviderConfigSpec{
					Credentials: namespacedv1beta1.NamespacedProviderCredentials{
						Source: xpv2.CredentialsSourceSecret,
						SecretRef: &xpv2.LocalSecretKeySelector{
							LocalSecretReference: xpv2.LocalSecretReference{Name: "creds"},
							Key:                  "credentials",
						},
					},
				},
			},
			want: namespacedv1beta1.ProviderConfigSpec{
				Credentials: namespacedv1beta1.ProviderCredentials{
					Source: xpv2.CredentialsSourceSecret,
					CommonCredentialSelectors: xpv2.CommonCredentialSelectors{
						SecretRef: &xpv2.SecretKeySelector{
							SecretReference: xpv2.SecretReference{Name: "creds", Namespace: exampleNamespace},
							Key:             "credentials",
						},
					},
				},
			},
		},
		"NilSecretRefStaysNil": {
			args: args{
				namespace: exampleNamespace,
				spec: namespacedv1beta1.NamespacedProviderConfigSpec{
					Credentials: namespacedv1beta1.NamespacedProviderCredentials{
						Source: xpv2.CredentialsSourceInjectedIdentity,
					},
				},
			},
			want: namespacedv1beta1.ProviderConfigSpec{
				Credentials: namespacedv1beta1.ProviderCredentials{
					Source: xpv2.CredentialsSourceInjectedIdentity,
				},
			},
		},
		"EndpointFieldsPassThrough": {
			args: args{
				namespace: exampleNamespace,
				spec: namespacedv1beta1.NamespacedProviderConfigSpec{
					Endpoint:    ptr.To("https://api.example.test/"),
					S3Endpoint:  ptr.To("https://s3.example.test"),
					HTTPTimeout: ptr.To("30s"),
					Credentials: namespacedv1beta1.NamespacedProviderCredentials{
						Source: xpv2.CredentialsSourceNone,
					},
				},
			},
			want: namespacedv1beta1.ProviderConfigSpec{
				Endpoint:    ptr.To("https://api.example.test/"),
				S3Endpoint:  ptr.To("https://s3.example.test"),
				HTTPTimeout: ptr.To("30s"),
				Credentials: namespacedv1beta1.ProviderCredentials{
					Source: xpv2.CredentialsSourceNone,
				},
			},
		},
		"FsAndEnvPassThrough": {
			args: args{
				namespace: exampleNamespace,
				spec: namespacedv1beta1.NamespacedProviderConfigSpec{
					Credentials: namespacedv1beta1.NamespacedProviderCredentials{
						Source: xpv2.CredentialsSourceFilesystem,
						Fs:     &xpv2.FsSelector{Path: "/creds"},
						Env:    &xpv2.EnvSelector{Name: "PROVIDER_CREDS"},
					},
				},
			},
			want: namespacedv1beta1.ProviderConfigSpec{
				Credentials: namespacedv1beta1.ProviderCredentials{
					Source: xpv2.CredentialsSourceFilesystem,
					CommonCredentialSelectors: xpv2.CommonCredentialSelectors{
						Fs:  &xpv2.FsSelector{Path: "/creds"},
						Env: &xpv2.EnvSelector{Name: "PROVIDER_CREDS"},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := resolveNamespacedSpec(tc.args.spec, tc.args.namespace)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("resolveNamespacedSpec() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

const (
	exampleToken     = "CW-SECRET-example"
	exampleNamespace = "team-a"
)

func TestBuildConfiguration(t *testing.T) {
	type args struct {
		creds map[string]any
		spec  *namespacedv1beta1.ProviderConfigSpec
	}

	cases := map[string]struct {
		args    args
		want    map[string]any
		wantErr bool
	}{
		"TokenOnly": {
			args: args{
				creds: map[string]any{keyToken: exampleToken},
				spec:  &namespacedv1beta1.ProviderConfigSpec{},
			},
			want: map[string]any{keyToken: exampleToken},
		},
		"SpecFieldsPassThrough": {
			args: args{
				creds: map[string]any{keyToken: exampleToken},
				spec: &namespacedv1beta1.ProviderConfigSpec{
					Endpoint:    ptr.To("https://api.example.test/"),
					S3Endpoint:  ptr.To("https://s3.example.test"),
					HTTPTimeout: ptr.To("30s"),
				},
			},
			want: map[string]any{
				keyToken:       exampleToken,
				keyEndpoint:    "https://api.example.test/",
				keyS3Endpoint:  "https://s3.example.test",
				keyHTTPTimeout: "30s",
			},
		},
		"EmptySpecFieldsAndUnknownCredentialKeysDropped": {
			args: args{
				creds: map[string]any{keyToken: exampleToken, keyEndpoint: "https://ignored.example.test/"},
				spec:  &namespacedv1beta1.ProviderConfigSpec{Endpoint: ptr.To("")},
			},
			want: map[string]any{keyToken: exampleToken},
		},
		"MissingToken": {
			args: args{
				creds: map[string]any{},
				spec:  &namespacedv1beta1.ProviderConfigSpec{Endpoint: ptr.To("https://api.example.test/")},
			},
			wantErr: true,
		},
		"NonStringToken": {
			args: args{
				creds: map[string]any{keyToken: 42},
				spec:  &namespacedv1beta1.ProviderConfigSpec{},
			},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := buildConfiguration(tc.args.creds, tc.args.spec)
			if (err != nil) != tc.wantErr {
				t.Fatalf("buildConfiguration() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("buildConfiguration() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
