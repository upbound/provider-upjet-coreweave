# provider-coreweave

`provider-coreweave` is a [Crossplane](https://crossplane.io/) provider for
CoreWeave, built with [Upjet](https://github.com/crossplane/upjet) on top of
the CoreWeave Terraform provider.

Every resource is available in two flavors: cluster-scoped
(`*.coreweave.upbound.io`) and namespaced
(`*.coreweave.m.upbound.io`).

## Getting started

Install the provider with [`examples/install.yaml`](examples/install.yaml),
create a credentials Secret and a ProviderConfig as shown in
[`examples/cluster/providerconfig/`](examples/cluster/providerconfig/) or
[`examples/namespaced/providerconfig/`](examples/namespaced/providerconfig/),
then apply any example under `examples/`.

## Developing

```console
make generate   # regenerate APIs, controllers, CRDs and examples from the Terraform provider
make reviewable # generate, lint and test; run before opening a pull request
make run        # run the provider out-of-cluster against the current kubeconfig
make e2e        # deploy to a kind cluster and run the uptest end-to-end tests
```

See [CLAUDE.md](CLAUDE.md) for the repository layout and the steps to add a resource.

The upstream Terraform provider does not export its constructor yet, so the Go
module is consumed from the Upbound fork: `go.mod` replaces
`github.com/coreweave/terraform-provider-coreweave` with a pseudo-version of
`github.com/upbound/terraform-provider-coreweave` on the `upjet-v0.26.0` branch,
which adds the `xpprovider` package.

## Reporting issues

Please open an [issue](https://github.com/upbound/provider-upjet-coreweave/issues)
for bug reports, feature requests, or questions.
