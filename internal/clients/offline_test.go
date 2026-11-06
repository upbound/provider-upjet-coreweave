// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	networkingv1beta1 "buf.build/gen/go/coreweave/networking/protocolbuffers/go/coreweave/networking/v1beta1"
	"github.com/coreweave/terraform-provider-coreweave/coreweave"
	"github.com/coreweave/terraform-provider-coreweave/xpprovider"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	rtfake "github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/crossplane/upjet/v2/pkg/diffserver"
	"github.com/crossplane/upjet/v2/pkg/terraform"
	"github.com/google/go-cmp/cmp"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clusterv1beta1 "github.com/upbound/provider-coreweave/apis/cluster/v1beta1"
	namespacedv1beta1 "github.com/upbound/provider-coreweave/apis/namespaced/v1beta1"
)

const (
	// offlineTestHost is a host no test can reach: the tripwire armed by
	// TestMain must see a request for it when nothing stands in the way.
	offlineTestHost     = "api.coreweave.example"
	offlineTestEndpoint = "https://" + offlineTestHost + "/"
	offlineTestVPCID    = "1b5274f2-8012-4b68-9010-cc4c51613302"
)

// egress is the tripwire armed by TestMain, shared by every test in the
// package. Tests that read it must not run in parallel with one another.
var egress *proxyRecorder

// proxyRecorder is an HTTP proxy that records every request routed through it
// and forwards none of them. The CoreWeave client builds its transports with
// http.ProxyFromEnvironment, so pointing HTTPS_PROXY at the recorder puts a
// tripwire on the process's real network boundary: nothing is stubbed or
// injected, and a request cannot leave for the CoreWeave API without being
// recorded here first.
type proxyRecorder struct {
	mu   sync.Mutex
	seen []string
}

func (p *proxyRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CONNECT carries its target in Host; an absolute-form proxy request
	// carries it in URL.
	target := r.Host
	if target == "" && r.URL != nil {
		target = r.URL.Host
	}
	p.mu.Lock()
	p.seen = append(p.seen, r.Method+" "+target)
	p.mu.Unlock()
	http.Error(w, "blocked by the egress tripwire", http.StatusBadGateway)
}

func (p *proxyRecorder) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen = nil
}

// reached reports whether any recorded request targeted host.
func (p *proxyRecorder) reached(host string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range p.seen {
		if strings.HasSuffix(r, " "+host) {
			return true
		}
	}
	return false
}

func TestMain(m *testing.M) {
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot listen for the egress tripwire: %v\n", err)
		os.Exit(1)
	}
	egress = &proxyRecorder{}
	srv := &http.Server{Handler: egress, ReadHeaderTimeout: time.Second}
	go srv.Serve(l) //nolint:errcheck // the listener is closed when the process exits
	// net/http reads the proxy environment once, on the first request of the
	// process, so it must be set before any test sends one.
	for _, kv := range [][2]string{{"HTTPS_PROXY", "http://" + l.Addr().String()}, {"HTTP_PROXY", "http://" + l.Addr().String()}, {"NO_PROXY", ""}} {
		if err := os.Setenv(kv[0], kv[1]); err != nil {
			fmt.Fprintf(os.Stderr, "cannot set %s: %v\n", kv[0], err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

// staticTokenSource is the token source of the positive control: a client
// that is allowed to send.
type staticTokenSource string

func (s staticTokenSource) Token(context.Context) (string, error) { return string(s), nil }

// TestOfflineTerraformSetupBuilder asserts that the offline setup resolves the
// ProviderConfig a managed resource references, honours everything in it
// except the credentials, which reference a Secret no test creates, and wraps
// the upstream provider.
func TestOfflineTerraformSetupBuilder(t *testing.T) {
	const (
		clusterEndpoint    = "https://cluster." + offlineTestHost + "/"
		namespacedEndpoint = "https://team-a." + offlineTestHost + "/"
		legacyEndpoint     = "https://legacy." + offlineTestHost + "/"
		s3Endpoint         = "https://s3." + offlineTestHost
		httpTimeout        = "30s"
		mrNamespace        = "team-a"
	)
	credentials := namespacedv1beta1.ProviderCredentials{
		Source: xpv2.CredentialsSourceSecret,
		CommonCredentialSelectors: xpv2.CommonCredentialSelectors{
			SecretRef: &xpv2.SecretKeySelector{
				SecretReference: xpv2.SecretReference{Name: "missing-creds", Namespace: "crossplane-system"},
				Key:             "credentials",
			},
		},
	}
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{clusterv1beta1.SchemeBuilder.AddToScheme, namespacedv1beta1.SchemeBuilder.AddToScheme} {
		if err := add(scheme); err != nil {
			t.Fatalf("cannot build the test scheme: %v", err)
		}
	}
	kc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&namespacedv1beta1.ClusterProviderConfig{
			ObjectMeta: metav1.ObjectMeta{Name: defaultProviderConfigName},
			Spec: namespacedv1beta1.ProviderConfigSpec{
				Endpoint: ptr.To(clusterEndpoint), S3Endpoint: ptr.To(s3Endpoint), HTTPTimeout: ptr.To(httpTimeout), Credentials: credentials,
			},
		},
		&namespacedv1beta1.ProviderConfig{
			ObjectMeta: metav1.ObjectMeta{Name: defaultProviderConfigName, Namespace: mrNamespace},
			Spec: namespacedv1beta1.NamespacedProviderConfigSpec{
				Endpoint: ptr.To(namespacedEndpoint),
				Credentials: namespacedv1beta1.NamespacedProviderCredentials{
					Source:    xpv2.CredentialsSourceSecret,
					SecretRef: &xpv2.LocalSecretKeySelector{LocalSecretReference: xpv2.LocalSecretReference{Name: "missing-creds"}, Key: "credentials"},
				},
			},
		},
		&clusterv1beta1.ProviderConfig{
			ObjectMeta: metav1.ObjectMeta{Name: defaultProviderConfigName},
			Spec: clusterv1beta1.ProviderConfigSpec{
				Endpoint: ptr.To(legacyEndpoint),
				Credentials: clusterv1beta1.ProviderCredentials{
					Source:                    xpv2.CredentialsSourceSecret,
					CommonCredentialSelectors: credentials.CommonCredentialSelectors,
				},
			},
		},
	).Build()

	modern := func(kind string) xpresource.Managed {
		mg := &rtfake.ModernManaged{ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: mrNamespace, UID: "test-uid"}}
		mg.SetProviderConfigReference(&xpv2.ProviderConfigReference{Kind: kind, Name: defaultProviderConfigName})
		return mg
	}
	legacy := func() xpresource.Managed {
		mg := &rtfake.LegacyManaged{ObjectMeta: metav1.ObjectMeta{Name: "example", UID: "test-uid"}}
		mg.SetProviderConfigReference(&xpv2.Reference{Name: defaultProviderConfigName})
		return mg
	}

	cases := map[string]struct {
		mg   xpresource.Managed
		want terraform.ProviderConfiguration
	}{
		"ClusterProviderConfigFromNamespacedResource": {
			mg:   modern(namespacedv1beta1.ClusterProviderConfigKind),
			want: terraform.ProviderConfiguration{keyToken: offlineToken, keyEndpoint: clusterEndpoint, keyS3Endpoint: s3Endpoint, keyHTTPTimeout: httpTimeout},
		},
		"ProviderConfigFromOwnNamespace": {
			mg:   modern(namespacedv1beta1.ProviderConfigKind),
			want: terraform.ProviderConfiguration{keyToken: offlineToken, keyEndpoint: namespacedEndpoint},
		},
		"LegacyProviderConfig": {
			mg:   legacy(),
			want: terraform.ProviderConfiguration{keyToken: offlineToken, keyEndpoint: legacyEndpoint},
		},
	}

	upstream, err := xpprovider.GetProvider(context.Background(), "test")
	if err != nil {
		t.Fatalf("cannot build the upstream provider: %v", err)
	}
	setup := OfflineTerraformSetupBuilder(upstream)
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := setup(context.Background(), kc, tc.mg)
			if err != nil {
				t.Fatalf("offline setup failed: %v", err)
			}
			if diff := cmp.Diff(tc.want, got.Configuration); diff != "" {
				t.Errorf("unexpected provider configuration, -want +got:\n%s", diff)
			}
			offline, ok := got.FrameworkProvider.(*offlineProvider)
			if !ok || offline.Provider != upstream {
				t.Errorf("expected the upstream provider wrapped by offlineProvider, got %T", got.FrameworkProvider)
			}
		})
	}
}

// configureOffline runs the offline provider's Configure with the given
// endpoint, the way the Terraform plugin framework does, and returns the
// client it hands the resources.
func configureOffline(t *testing.T, endpoint string) *coreweave.Client {
	t.Helper()
	ctx := context.Background()
	upstream, err := xpprovider.GetProvider(ctx, "test")
	if err != nil {
		t.Fatalf("cannot build the upstream provider: %v", err)
	}
	p := &offlineProvider{Provider: upstream}
	schemaResp := fwprovider.SchemaResponse{}
	p.Schema(ctx, fwprovider.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), map[string]tftypes.Value{
		keyEndpoint:    tftypes.NewValue(tftypes.String, endpoint),
		keyS3Endpoint:  tftypes.NewValue(tftypes.String, nil),
		keyToken:       tftypes.NewValue(tftypes.String, offlineToken),
		keyHTTPTimeout: tftypes.NewValue(tftypes.String, "5s"),
	})
	resp := fwprovider.ConfigureResponse{}
	p.Configure(ctx, fwprovider.ConfigureRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("offline Configure failed: %v", resp.Diagnostics)
	}
	c, ok := resp.ResourceData.(*coreweave.Client)
	if !ok || c == nil {
		t.Fatalf("expected a *coreweave.Client as resource data, got %T", resp.ResourceData)
	}
	if resp.DataSourceData != resp.ResourceData {
		t.Errorf("expected data sources and resources to share the client")
	}
	return c
}

// TestOfflineProviderBlocksEgress asserts that a call on the client the
// offline provider hands out fails as a diff the server cannot compute, and
// that nothing reaches the process's network boundary.
func TestOfflineProviderBlocksEgress(t *testing.T) {
	c := configureOffline(t, offlineTestEndpoint)
	egress.reset()

	_, err := c.GetVPC(coreweave.WithoutRetries(context.Background()), &networkingv1beta1.GetVPCRequest{Id: offlineTestVPCID})
	if !diffserver.IsDiffComputationNotSupportedError(err) {
		t.Fatalf("expected the call to fail as unsupported offline, got: %v", err)
	}
	if !strings.Contains(err.Error(), "GetVPC") {
		t.Errorf("expected the error to name the blocked procedure, got: %v", err)
	}
	if egress.reached(offlineTestHost + ":443") {
		t.Fatalf("a request for %s reached the network boundary", offlineTestHost)
	}
}

// TestEgressTripwireSeesAnUnguardedClient is the positive control of the
// tripwire: the same call through a client built without the guard, and
// with a token to send, is recorded at the network boundary.
func TestEgressTripwireSeesAnUnguardedClient(t *testing.T) {
	c, err := coreweave.NewClientWithOptions(offlineTestEndpoint, defaultS3Endpoint, time.Second, staticTokenSource("CW-SECRET-test"), "test", coreweave.ClientOptions{})
	if err != nil {
		t.Fatalf("cannot build the unguarded client: %v", err)
	}
	egress.reset()

	if _, err := c.GetVPC(coreweave.WithoutRetries(context.Background()), &networkingv1beta1.GetVPCRequest{Id: offlineTestVPCID}); err == nil {
		t.Fatalf("expected the tripwire to fail the call")
	}
	if !egress.reached(offlineTestHost + ":443") {
		t.Fatalf("expected a request for %s at the network boundary", offlineTestHost)
	}
}

// TestOfflineTokenSourceRefuses asserts that the token source never yields a
// credential, so that a request bypassing the connect client cannot carry one.
func TestOfflineTokenSourceRefuses(t *testing.T) {
	token, err := offlineTokenSource{}.Token(context.Background())
	if token != "" || !diffserver.IsDiffComputationNotSupportedError(err) {
		t.Fatalf("expected no token and an unsupported-offline error, got %q, %v", token, err)
	}
}

func TestParseTimeout(t *testing.T) {
	cases := map[string]struct {
		raw     string
		want    time.Duration
		wantErr bool
	}{
		"GoDuration":  {raw: "30s", want: 30 * time.Second},
		"BareSeconds": {raw: "15", want: 15 * time.Second},
		"Zero":        {raw: "0s", wantErr: true},
		"Garbage":     {raw: "soon", wantErr: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := parseTimeout(tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseTimeout(%q) error = %v, wantErr %v", tc.raw, err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseTimeout(%q) mismatch (-want +got):\n%s", tc.raw, diff)
			}
		})
	}
}
