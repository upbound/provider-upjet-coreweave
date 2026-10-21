# provider-coreweave - Architecture

A Crossplane provider for CoreWeave, built with [Upjet](https://github.com/crossplane/upjet)
on top of the CoreWeave Terraform provider. Upjet ingests the Terraform
provider's schema and docs and code-generates Crossplane CRDs and controllers.

This document describes only the project's structure and conventions so an agent
can quickly find the files and configuration involved in common tasks
(especially **adding a new resource**). It is not a usage guide.

## Key facts

- **Provider name:** `coreweave` (Makefile `PROVIDER_NAME`). Module path:
  `github.com/upbound/provider-coreweave`.
- **Upstream TF provider:** pinned in the `Makefile` (`TERRAFORM_PROVIDER_SOURCE`,
  `TERRAFORM_PROVIDER_REPO`, `TERRAFORM_PROVIDER_VERSION`). Its Go module is
  linked into the binary and driven in-process; there is no terraform binary or
  provider mirror in the runtime image.
- **Upstream constructor:** `terraform-provider-coreweave` keeps its constructor
  in `internal/provider`, so the module is consumed through the `xpprovider`
  package of the Upbound fork's `upjet-v0.26.0` branch: a `replace` directive
  in `go.mod` pins `github.com/upbound/terraform-provider-coreweave` to a
  pseudo-version of that branch. The upstream version is pinned in four places
  that must move together: `TERRAFORM_PROVIDER_VERSION` (Makefile), the module
  version and the `replace` pseudo-version (`go.mod`) and
  `upstreamProviderVersion` (`config/terraform_provider.go`).
- **Protobuf registration conflict:** the upstream sandbox client and the
  Kubernetes client libraries both register the gnostic OpenAPI v3 annotation
  extension, which protobuf-go treats as fatal at init. The `Makefile` sets the
  conflict policy to `warn`: an ldflag for built binaries and the
  `GOLANG_PROTOBUF_REGISTRATION_CONFLICT` environment variable for `go run`,
  `go generate` and `go test`. Running the generator or the provider outside
  make needs that variable set.
- **Credentials and endpoints:** the credentials secret is a JSON object with
  the `token` key only. The optional provider arguments are ProviderConfig
  spec fields in both scopes: `spec.endpoint`, `spec.s3Endpoint` and
  `spec.httpTimeout`. `internal/clients/coreweave.go` maps both onto the
  provider block.
- **Two plugin libraries:** resources implemented upstream with
  terraform-plugin-sdk/v2 are listed in `terraformPluginSDKExternalNameConfigs`,
  resources implemented with terraform-plugin-framework in
  `terraformPluginFrameworkExternalNameConfigs` (both in `config/external_name.go`).
  A resource belongs to exactly one list; the list decides which upjet client
  reconciles it. `config/terraform_provider.go` returns the upstream provider
  instance(s). The CoreWeave provider is terraform-plugin-framework only, so
  `sdkProvider()` returns nil and the SDK map stays empty.
- **Dual scope:** every resource is generated twice: once **cluster-scoped**
  (group suffix `.upbound.io`) and once **namespaced** (group suffix
  `.m.upbound.io`). Configuration is mirrored under `config/cluster/` and
  `config/namespaced/`.

## API groups

Groups follow the upstream service prefixes with the underscores removed;
kinds are the remaining words of the Terraform resource name, with
initialisms kept upper-case (`VPC`, `OIDCConfig`).

| API group            | Kinds                                                                                                                                | Terraform resources                        |
|----------------------|--------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------|
| `cks`                | Cluster                                                                                                                              | `coreweave_cks_cluster`                    |
| `containerregistry`  | Namespace, AccessConfiguration, LifecyclePolicy                                                                                      | `coreweave_container_registry_*`           |
| `inference`          | CapacityClaim, Deployment, Gateway                                                                                                   | `coreweave_inference_*`                    |
| `networking`         | VPC                                                                                                                                  | `coreweave_networking_vpc`                 |
| `objectstorage`      | Bucket, BucketInventory, BucketLifecycleConfiguration, BucketPolicy, BucketSettings, BucketVersioning, AccessKey, OrganizationAccessPolicy | `coreweave_object_storage_*`               |
| `sandbox`            | ManagedRunner                                                                                                                        | `coreweave_sandbox_managed_runner`         |
| `workloadfederation` | OIDCConfig                                                                                                                           | `coreweave_workload_federation_oidc_config` |

External names, by the shape of the upstream identifier
(`config/external_name.go`):

- **Server-assigned UUID `id`** (Cluster, CapacityClaim, Deployment, Gateway,
  OIDCConfig, VPC, AccessKey): `FrameworkResourceWithComputedIdentifier` with
  the nil UUID as the pre-creation placeholder.
- **No `id` attribute, identified by `name`** (Bucket, OrganizationAccessPolicy):
  `frameworkNameAsIdentifier`, so `name` is the external name and is not a
  spec field.
- **No `id` attribute, identified by the bucket** (BucketInventory also by its
  own `name`): `externalNameFromArguments`; the external name is assigned after
  creation in the upstream import format, so `bucket` stays a referenceable
  spec field.
- **Canonical API names** (`namespaces/<name>` and its `/accessConfiguration`
  and `/lifecyclePolicy` singletons): Namespace uses
  `TemplatedStringAsIdentifier` with the namespace name as external name; the
  singletons take the canonical name after the first read and keep `namespace`
  referenceable.
- **Operator-assigned id** (ManagedRunner): `ParameterAsIdentifier("runner_id")`,
  so the external name is the runner id.

## Directory layout

```
config/                       Upjet provider configuration (hand-written): the heart of resource setup
  terraform_provider.go       sdkProvider() / frameworkProvider(): the upstream provider constructors
  external_name.go            SDK and framework external-name maps: TF resource -> external-name strategy
  groups.go                   GroupMap + ReplaceGroupWords: TF resource -> (ShortGroup, Kind)
  provider.go                 GetProvider() (cluster); embeds schema.json + provider-metadata.yaml
  provider_namespaced.go      GetProviderNamespaced()
  generated.lst               JSON array of TF resource names with a generated CRD (kept in sync by codegen)
  schema.json                 Scraped TF provider schema (generated; input to codegen)
  provider-metadata.yaml      Scraped TF docs: per-resource examples + argument docs (input to codegen)
  templates/controller.go.tmpl  Controller source rendered per resource; adds the ReconciliationPolicy wiring
  cluster/
    provider.go               init(): registers each group's Configure into ProviderConfiguration
    configuration.go          Configurator registry type + global ProviderConfiguration var
    <group>/config.go         Per-group AddResourceConfigurator calls: cross-resource references live here
  namespaced/                 Mirror of cluster/ for the namespaced provider (<group>/config.go is duplicated)

apis/                         Go API types. Hand-written: {cluster,namespaced}/v1beta1 (ProviderConfig)
  cluster/<group>/<version>/  Generated zz_<resource>_types.go, zz_<resource>_terraformed.go, zz_generated.*.go
  namespaced/<group>/<version>/ (mirror)

internal/
  controller/{cluster,namespaced}/<group>/<resource>/   Generated controllers (one dir per resource)
  controller/{cluster,namespaced}/providerconfig/       Hand-written ProviderConfig controllers
  controller/{cluster,namespaced}/zz_setup.go           Generated controller registration
  clients/                    TerraformSetupBuilder: ProviderConfig -> credentials -> provider configuration
  features/                   Feature flags

cmd/
  generator/main.go           Codegen entrypoint (run by `make generate`)
  provider/main.go            Hand-written provider binary entrypoint

hack/                         boilerplate.go.txt (license header injected into generated code), prepare.sh (scaffold rename)
package/crds/                 Generated CRD YAML (one file per resource per scope)
examples/                     Hand-curated examples (the source of truth shipped to users / used by uptest)
examples-generated/           Codegen output from provider-metadata.yaml examples (copy into examples/, then edit)
.work/                        Scratch: cloned TF provider docs, terraform workdir, schema generation
```

## Generation pipeline (`make generate`)

1. `generate.init` runs `$(TERRAFORM_PROVIDER_SCHEMA)`: installs terraform 1.5.7,
   `terraform providers schema -json`, writes `config/schema.json`.
2. `pull-docs` sparse-checks-out `$(TERRAFORM_DOCS_PATH)` from the upstream TF
   provider repo at the pinned tag into `.work/<source>/`.
3. `apis/generate.go` scrapes the docs into `config/provider-metadata.yaml`,
   runs `cmd/generator` (which loads `GetProvider` + `GetProviderNamespaced`,
   applies the config in `config/`, and writes `apis/`, `internal/controller/`,
   `package/crds/` and `examples-generated/`), then controller-gen and angryjet.

A resource only gets an `examples-generated/` file if it has an `examples:`
block in `provider-metadata.yaml`; otherwise build the example from the CRD schema.

## Resource-configuration knobs (where to edit when adding a resource)

Adding a TF resource requires touching, in order:

1. **`config/external_name.go`**: add to the map matching the plugin library the
   upstream provider implements the resource with. The maps drive both the
   external-name behavior and the upjet include lists (only listed resources are
   generated). Typical choices: `config.IdentifierFromProvider` for SDKv2
   resources whose id the API assigns,
   `config.FrameworkResourceWithComputedIdentifier("id", "<placeholder>")` for
   framework resources, `config.NameAsIdentifier` when the name is the id.

2. **`config/groups.go`**: add to `GroupMap` when the default derivation (first
   word after the prefix is the group, the rest is the Kind) is wrong.
   `ReplaceGroupWords(group, count)` strips the `coreweave_` prefix, drops the
   first `count` words and camel-cases the remainder into the Kind.

3. **`config/cluster/<group>/config.go` AND `config/namespaced/<group>/config.go`**
   (keep them identical) - add an `AddResourceConfigurator` block **only if the
   resource has ID references to other resources**, and register the group's
   `Configure` in both `provider.go` files. Each reference:
   ```go
   r.References["<dotted_snake_case_tf_path>"] = config.Reference{
       TerraformName: "coreweave_<target>",
       Extractor:     `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("id",true)`,
   }
   ```
   Reference keys are the **dotted snake_case Terraform attribute paths**
   (e.g. `network_id`, `ipv4_private_pools.pools.id`). Do **not** use `[*]` for
   list elements: Upjet silently drops such references.

After editing config, run `make generate`, then curate `examples/`.

## Naming and convention reference

- **Group/Kind**: `<group>.coreweave.upbound.io` (cluster) and
  `<group>.coreweave.m.upbound.io` (namespaced); version is `v1beta1` (set by
  `ExternalNameConfigurations`).
- **Examples**: `examples/<scope>/<group>/<version>/<kind-lowercase>.yaml`.
  Cross-resource fields use `<field>Selector.matchLabels` against
  `testing.upbound.io/example-name`, whose value is the file's
  `meta.upbound.io/example-id` with `/` replaced by `-` (e.g.
  `vpc-v1beta1-subnet`) so files applied together never cross-select.
  Two resources of the same kind in one file link by `<field>Ref.name`
  instead of a selector. Every managed resource that sets
  `spec.forProvider.name` uses the same value as its `metadata.name`.
- **Credentials**: the ProviderConfig secret holds a JSON object with the
  `token` key; the remaining provider block arguments are ProviderConfig spec
  fields, both mapped in `internal/clients/coreweave.go`.

## E2E testing

`make uptest` / `make e2e` drive examples through a live control plane.
`cluster/test/setup.sh` creates the credentials secret and a default
ProviderConfig of each scope. `UPTEST_DATASOURCE_PATH` injects dynamic values.
On a pull request, a maintainer comment `/test-examples="examples/cluster/<group>/v1beta1/<kind>.yaml"`
runs the same flow in CI.
