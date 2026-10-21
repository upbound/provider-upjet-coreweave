package clients

import (
	"context"
	"testing"

	rtfake "github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

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
	exampleToken              = "CW-SECRET-example"
	exampleNamespace          = "team-a"
	defaultProviderConfigName = "default"
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

// TestResolveModern asserts that a namespaced managed resource resolves a
// ClusterProviderConfig without its own namespace, and a ProviderConfig from
// its own namespace. The fake client keys objects on their exact namespace,
// like the diff server's in-memory store, so the first case fails without
// the fix in resolveModern.
func TestResolveModern(t *testing.T) {
	const (
		clusterEndpoint    = "https://cluster.api.example.test/"
		namespacedEndpoint = "https://team-a.api.example.test/"
	)
	scheme := runtime.NewScheme()
	if err := namespacedv1beta1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("cannot build the test scheme: %v", err)
	}

	type want struct {
		endpoint string
		err      bool
	}
	cases := map[string]struct {
		kind string
		want want
	}{
		"ClusterProviderConfigFromNamespacedResource": {
			kind: namespacedv1beta1.ClusterProviderConfigKind,
			want: want{endpoint: clusterEndpoint},
		},
		"ProviderConfigFromOwnNamespace": {
			kind: namespacedv1beta1.ProviderConfigKind,
			want: want{endpoint: namespacedEndpoint},
		},
		"UnknownKind": {
			kind: "NoSuchProviderConfig",
			want: want{err: true},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			kc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&namespacedv1beta1.ClusterProviderConfig{
					ObjectMeta: metav1.ObjectMeta{Name: defaultProviderConfigName},
					Spec:       namespacedv1beta1.ProviderConfigSpec{Endpoint: ptr.To(clusterEndpoint)},
				},
				&namespacedv1beta1.ProviderConfig{
					ObjectMeta: metav1.ObjectMeta{Name: defaultProviderConfigName, Namespace: exampleNamespace},
					Spec:       namespacedv1beta1.NamespacedProviderConfigSpec{Endpoint: ptr.To(namespacedEndpoint)},
				},
			).Build()
			mg := &rtfake.ModernManaged{ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: exampleNamespace, UID: "test-uid"}}
			mg.SetProviderConfigReference(&xpv2.ProviderConfigReference{Kind: tc.kind, Name: defaultProviderConfigName})

			got, err := resolveModern(context.Background(), kc, mg)
			if (err != nil) != tc.want.err {
				t.Fatalf("resolveModern() error = %v, wantErr %v", err, tc.want.err)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tc.want.endpoint, ptr.Deref(got.Endpoint, "")); diff != "" {
				t.Errorf("resolveModern() resolved the wrong ProviderConfig (-want +got):\n%s", diff)
			}
		})
	}
}
