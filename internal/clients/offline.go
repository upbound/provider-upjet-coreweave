// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"time"

	"connectrpc.com/connect/v2"
	"github.com/coreweave/terraform-provider-coreweave/coreweave"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/upjet/v2/pkg/diffserver"
	"github.com/crossplane/upjet/v2/pkg/terraform"
	"github.com/hashicorp/terraform-plugin-framework/function"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// offlineToken is the placeholder the offline setup configures the
	// provider with in place of the API token. It is never sent: the client
	// the offline provider builds refuses every call before it leaves the
	// process, and its token source refuses to produce a token at all.
	offlineToken = "CW-SECRET-offline-diff-server-placeholder" //nolint:gosec // a placeholder, never sent

	offlineUserAgent = "provider-coreweave offline diff server"

	// The defaults of the upstream provider for the endpoints and the API
	// timeout, applied when the ProviderConfig leaves them unset.
	defaultEndpoint    = "https://api.coreweave.com/"
	defaultS3Endpoint  = "https://cwobject.com"
	defaultHTTPTimeout = 10 * time.Second

	errOfflineTokenRequested    = "the offline diff server refused to produce an API token: computing this diff requires calling the CoreWeave API, which is not possible offline"
	fmtErrOfflineRequestBlocked = "the offline diff server blocked an outbound CoreWeave API call (%s): computing this diff requires calling the CoreWeave API, which is not possible offline"
)

// OfflineTerraformSetupBuilder returns a Terraform setup that configures the
// CoreWeave Terraform provider without credentials and without contacting the
// CoreWeave API. Use it in place of TerraformSetupBuilder in the diff server
// only.
//
// The ProviderConfig is still resolved, so that the endpoints and the HTTP
// timeout it carries are honoured the same way they are for a live reconcile;
// only its credentials are ignored, and the Secret they would come from is
// never read. The returned setup carries the upstream provider wrapped by
// offlineProvider, whose Configure hands every resource a client that cannot
// send a request.
func OfflineTerraformSetupBuilder(fwProvider fwprovider.Provider) terraform.SetupFn {
	offline := &offlineProvider{Provider: fwProvider}
	return func(ctx context.Context, c client.Client, mg resource.Managed) (terraform.Setup, error) {
		pcSpec, err := resolveProviderConfig(ctx, c, mg)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, errResolveProviderConfig)
		}
		cfg, err := buildConfiguration(map[string]any{keyToken: offlineToken}, pcSpec)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, errBuildConfiguration)
		}
		return terraform.Setup{
			Configuration:     cfg,
			FrameworkProvider: offline,
		}, nil
	}
}

// offlineProvider is the upstream CoreWeave Terraform provider with its
// Configure replaced. The upstream Configure builds an API client from the
// configuration block and the COREWEAVE_* environment variables, which take
// precedence over the block; this one never reads the environment and builds
// a client that refuses every call. Everything else, the schema and the
// resources in particular, is the upstream provider's own.
type offlineProvider struct {
	fwprovider.Provider
}

// offlineProviderModel mirrors the upstream provider's configuration block.
type offlineProviderModel struct {
	Endpoint    types.String `tfsdk:"endpoint"`
	S3Endpoint  types.String `tfsdk:"s3_endpoint"`
	Token       types.String `tfsdk:"token"`
	HTTPTimeout types.String `tfsdk:"http_timeout"`
}

// Configure hands the resources a client that cannot reach the CoreWeave API.
func (p *offlineProvider) Configure(ctx context.Context, req fwprovider.ConfigureRequest, resp *fwprovider.ConfigureResponse) {
	var data offlineProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c, err := newOfflineClient(data)
	if err != nil {
		resp.Diagnostics.AddError("cannot build the offline CoreWeave client", err.Error())
		return
	}
	resp.DataSourceData = c
	resp.ResourceData = c
}

// Functions serves the upstream provider's functions, if it has any.
func (p *offlineProvider) Functions(ctx context.Context) []func() function.Function {
	if fp, ok := p.Provider.(fwprovider.ProviderWithFunctions); ok {
		return fp.Functions(ctx)
	}
	return nil
}

// newOfflineClient builds the CoreWeave client the resources expect in their
// provider data, from the configuration block alone, with the upstream
// defaults for what the block leaves unset. The client is fully functional up
// to the point where a call would leave the process: offlineGuard refuses
// every RPC, and the S3 client is only ever built after an RPC that mints the
// S3 credentials, so it is never built.
func newOfflineClient(data offlineProviderModel) (*coreweave.Client, error) {
	timeout := defaultHTTPTimeout
	if raw := data.HTTPTimeout.ValueString(); raw != "" {
		d, err := parseTimeout(raw)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot parse http_timeout %q", raw)
		}
		timeout = d
	}
	return coreweave.NewClientWithOptions(
		valueOr(data.Endpoint, defaultEndpoint),
		valueOr(data.S3Endpoint, defaultS3Endpoint),
		timeout,
		offlineTokenSource{},
		offlineUserAgent,
		coreweave.ClientOptions{},
		offlineGuard(),
	)
}

// parseTimeout accepts a Go duration, or a bare number of seconds as the
// upstream provider does.
func parseTimeout(raw string) (time.Duration, error) {
	d, err := time.ParseDuration(raw)
	if err != nil {
		if seconds, errSeconds := time.ParseDuration(raw + "s"); errSeconds == nil {
			d = seconds
		} else {
			return 0, err
		}
	}
	if d <= 0 {
		return 0, errors.New("duration must be greater than zero")
	}
	return d, nil
}

func valueOr(v types.String, fallback string) string {
	if s := v.ValueString(); s != "" {
		return s
	}
	return fallback
}

// offlineGuard is a connect interceptor that refuses every RPC, failing the
// diff at once with an error the diff server reports as
// DIFF_COMPUTATION_NOT_SUPPORTED, instead of letting the call reach the
// CoreWeave API or fail slowly against an unreachable endpoint. Every
// CoreWeave API call of the provider goes through the connect client it
// wraps, including the one that mints the credentials of the S3 client.
func offlineGuard() connect.ClientInterceptor {
	return func(connect.ClientFunc) connect.ClientFunc {
		return func(_ context.Context, spec connect.Spec) (connect.ClientStream, error) {
			return nil, diffserver.NewDiffComputationNotSupportedError(errors.Errorf(fmtErrOfflineRequestBlocked, spec.Procedure))
		}
	}
}

// offlineTokenSource refuses to produce a token, so that a request which
// reaches the authenticating transport without going through the connect
// client, such as a raw HTTP call of a data source, fails before it is sent
// rather than carrying a credential anywhere.
type offlineTokenSource struct{}

func (offlineTokenSource) Token(context.Context) (string, error) {
	return "", diffserver.NewDiffComputationNotSupportedError(errors.New(errOfflineTokenRequested))
}
