# FireFly

[Hyperledger FireFly](https://hyperledger.github.io/firefly/) is an implementation of a [multi-party system](https://github.com/hyperledger/firefly#multi-party-systems)
that simplifies data orchestration on top of blockchain and other peer-to-peer technologies.

This chart bootstraps a FireFly deployment on a [Kubernetes](https://kubernetes.io/) cluster using the [Helm](https://helm.sh/)
package manager. It can be used to deploy a FireFly node for a single organization within a multi-party system.

### Table of Contents

* [Prerequisites](#prerequisites)
* [Get Repo Info](#get-repo-info)
* [Install Chart](#install-chart)
* [Uninstall Chart](#uninstall-chart)
* [Upgrading Chart](#upgrading-chart)
* [Using as a Dependency](#using-as-a-dependency)
* [Deployment Architecture](#deployment-architecture)
* [Configuration](#configuration)
  * [Configuration File Templating](#configuration-file-templating)
  * [Additional Environment Variables](#additional-environment-variables)
  * [Ethereum](#ethereum)
    * [Smart Contract Deployment](#smart-contract-deployment)
  * [Fabric](#fabric)
    * [Chaincode](#chaincode)
    * [Identity Management](#identity-management)
  * [Ingress Example](#ingress-example)
  * [Database Migrations](#database-migrations)
  * [Auto-Registration](#auto-registration)
  * [DataExchange HTTPS and cert-manager](#dataexchange-https-and-cert-manager)
  * [Tokens via ERC1155 Connector](#tokens-via-erc1155-connector)
  * [Prometheus Support](#prometheus-support)
* [Automated Deployments](#automated-deployments)
  * [GitOps](#gitops)
    * [Flux V2](#flux-v2)
    * [ArgoCD](#argocd)
  * [Terraform](#terraform)



## Prerequisites

* Kubernetes 1.18+
* Helm 3.7+
* PV provisioner support in the underlying infrastructure
* _Recommended:_ cert-manager 1.4+

## Get Repo Info

Helm's [experimental OCI registry support](https://helm.sh/docs/topics/registries/) is used for publishing and retrieving
the FireFly chart, as a result one must log into [GHCR](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
to download the chart:

```shell
export HELM_EXPERIMENTAL_OCI=1

helm registry login ghcr.io
```

> **NOTE**: it is recommended to use a [GitHub personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
> when authenticating to the GHCR registry as opposed to using your GitHub password.

## Install Chart

```shell
helm install [RELEASE_NAME] --version 0.0.1 oci://ghcr.io/hyperledger/helm/firefly
```

_See [configuration](#Configuration) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

## Uninstall Chart

```shell
helm uninstall [RELEASE_NAME]
```

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Chart

```shell
helm upgrade [RELEASE_NAME] --install --version 0.0.2 oci://ghcr.io/hyperledger/helm/firefly
```

_See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) for command documentation._

## Using as a Dependency

You can also use the FireFly chart within your own parent chart's `Chart.yaml`:

```yaml
dependencies:
  # ...
  - name: firefly
    repository: "oci://ghcr.io/hyperledger/helm/"
    version: 0.0.1
```

Then download the chart dependency into your parent chart:

```shell
helm dep up path/to/parent-chart
```

_See [helm dependency](https://helm.sh/docs/helm/helm_dependency/) for command documentation._


## Deployment Architecture

FireFly provides a REST API with an event-driven paradigm that makes building multi-party interactions via
decentralized applications simpler. In order to do so, FireFly leverages extensible [connector plugins](https://hyperledger.github.io/firefly/architecture/plugin_architecture.html) that enable
swapping out the underlying blockchain and off-chain infrastructure easily.

As a result, a [FireFly node](https://hyperledger.github.io/firefly/architecture/node_component_architecture.html)
has several infrastructural dependencies:

* Blockchain connector (either Fabconnect -> Fabric, or Ethconnect -> Ethereum) for a [_private_ blockchain](https://hyperledger.github.io/firefly/keyconcepts/blockchain_protocols.html)
* A Fabric chaincode or Ethereum smart contract deployed to the underlying blockchain
* [Private data exchange](https://hyperledger.github.io/firefly/keyconcepts/data_exchange.html) (HTTPS + mTLS)
* Database (PostgreSQL)
* [Shared storage](https://hyperledger.github.io/firefly/keyconcepts/broadcast.html#shared-data) (IPFS)
* Optional tokens connector (ERC1155)

<p align="center">
  <img src="./../../../images/helm_chart_deployment_architecture.jpg" width="75%" />
</p>

As depicted above, the chart only aims to provide a means for deploying FireFly core, and then optionally [FireFly Ethconnect](ttps://github.com/hyperledger/firefly-ethconnect), [FireFly Fabconnect](ttps://github.com/hyperledger/firefly-fabconnect),
[FireFly DataExchange HTTPS](https://github.com/hyperledger/firefly-dataexchange-https) and the [FireFly Tokens ERC1155](https://github.com/hyperledger/firefly-tokens-erc1155) microservices.

> **NOTE**: support for deploying Ethconnect, Fabconnect, and Tokens ERC1155 is under development and will be included
> as part of the chart for its `0.1.0` release.

All other infrastructural dependencies such as the blockchain, PostgreSQL, and IPFS are considered out of scope for the chart,
and must be pre-provisioned in order for FireFly to be fully functioning.

## Configuration

The following describes how to use the chart's values to configure various aspects of the FireFly deployment.

### Configuration File Templating

FireFly itself has a robust YAML configuration file (usually named `firefly.core`) powered by [Viper](https://github.com/spf13/viper)
that allows one to define all the necessary configuration for the FireFly server, and the underlying
connectors it will use.

The chart provides a top-level `config` value which then contains sub-values such as `postgresUrl`, `ipfsApiUrl`,
`organizationName`, `adminEnabled`, etc. These sub-values are meant to provide an opinionated, safe way of templating
the `firefly.core` file. Based on which values are set, it will correctly configure the various connector plugins as well
as determine if additional ports will be exposed such as the admin, debug, and metrics ports.

The following values are required in order for FireFly to startup correctly:
* `config.organizationName`
* `config.organizationKey`
* `config.postgresUrl`
* `config.ipfsApiUrl` and `config.ipfsGatewayUrl`
* either:
    * `config.ethconnectUrl` and `config.fireflyContractAddress`
    * or, `config.fabconnectUrl` and `config.fabconnectSigner`

You can find documentation regarding each of these values, as well as all the other `config` values,
in the comments of the default [`values.yaml`](values.yaml). You can see how the values are used for
templating the `firefly.core` file by looking at the `firefly.coreConfig` helper function in [`_helpers.tpl`](templates/_helpers.tpl).

> **NOTE**: although `config.dataexchangeUrl` is available, by default `dataexchange.enabled` is `true` which will
> deploy a DataExchange HTTPS and automatically configure FireFly to use it.

If you would rather customize the templating of the `firefly.core` with your own values, you can use `config.templateOverride`:

```yaml
config:
  templateOverride: |
    org:
      name: {{ .Values.global.myOrgNameValue }}
    # etc. ...
```

See [`config.go`](../../../internal/config/config.go) for all available FireFly configuration options.

### Additional Environment Variables

If there are configurations you want to set via your own `ConfigMaps` or `Secrets`, it is recommended to do so
via environment variables which can be provided with the `core.extraEnv` list value. FireFly will automatically override
its config via environment variables prefixed with `FIREFLY_`. For example, if you want to set to the config value
`log.level` you would set the env var `FIREFLY_LOG_LEVEL`.

For a more detailed example using `core.extraEnv`, one could provide basic auth credentials for IPFS from a `Secret`
like so:

```yaml
core:
  extraEnv:
    - name: FIREFLY_PUBLICSTORAGE_IPFS_API_AUTH_USERNAME
      valueFrom:
        secretKeyRef:
          name: my-ipfs-basic-auth
          key: username
    - name: FIREFLY_PUBLICSTORAGE_IPFS_API_AUTH_PASSWORD
      valueFrom:
        secretKeyRef:
          name: my-ipfs-basic-auth
          key: password
```

### Ethereum

Configuring FireFly to use an [Ethereum](https://ethereum.org/en/) blockchain such as [Geth](https://geth.ethereum.org/),
[Quorum](https://github.com/ConsenSys/quorum), or [Hyperledger Besu](https://www.hyperledger.org/use/besu) requires first
having an instance of [FireFly Ethconnect](https://github.com/hyperledger/firefly-ethconnect) deployed and connected to
the JSONRPC port of an Ethereum node in the underlying network.

As was noted in [Deployment Architecture](#deployment-architecture), the chart will include support for deploying Ethconnect
as part of its `0.1.0` release. See [#272](https://github.com/hyperledger/firefly/issues/272) to track its progress. For now,
you can either deploy Ethconnect yourself or use a cloud provider like [Kaleido](https://www.kaleido.io) which provides
Ethconnect alongside its Ethereum nodes.

Once you have an Ethconnect instance ready, FireFly then needs three pieces of configuration:

* `config.organizationKey`: the Ethereum address of the organization's wallet / key which will be used for signing transactions
* `config.ethconnectUrl`: the HTTP/S URL of the Ethconnect instance FireFly will use
* `config.fireflyContractAddress`: the Ethconnect URI representing the deployed FireFly smart contract i.e.
  `/instances/0x965b92929108df1c77c156ba73d00ca851dcd2e1`. See [Smart Contract Deployment](#smart-contract-deployment)
  for how to you can deploy the contract yourself.

These will enable the FireFly deployment to connect to the Ethereum blockchain and submit batch pin transactions via
its smart contract on behalf of the organization it's representing.

#### Smart Contract Deployment

Currently, the chart offers no way for one to manage the [FireFly smart contract](../../../smart_contracts/ethereum/solidity_firefly/contracts/Firefly.sol).
Instead, the chart assumes it is already pre-provisioned via Ethconnect by one of the organizations.

If you have the contract available as gateway contract on Ethconnect, you can then deploy it via the API:

```shell
curl -v \
 -X POST \
 -H 'Content-Type: application/json' \
 -d '{}' \
 "${ETHCONNECT_URL/gateways/${FF_CONTRACT_GATEWAY}?ff-from=${ORG_WALLET_ADDRESS}&ff-sync=true"
```

The JSON returned by the API will have the Ethereum address of the smart contract in the `address` field.

> **NOTE**: the FireFly smart contract only needs to be deployed by one organization within the blockchain
> network. All organizations within a FireFly network must use the same smart contract instance in order for
> transactions to work properly.

If the contract is not available as a gateway contract on your Ethconnect instance, see the
Ethconnect docs for [deploying a contract](https://github.com/hyperledger/firefly-ethconnect#yaml-to-deploy-a-contract).

### Fabric

Configuring FireFly to use a [Hyperledger Fabric](https://www.hyperledger.org/use/fabric) blockchain requires first
having an instance of [FireFly Fabconnect](https://github.com/hyperledger/firefly-fabconnect) deployed and connected to
the gRPC port of a Fabric peer in the underlying network.

As was noted in [Deployment Architecture](#deployment-architecture), the chart will include support for deploying Fabconnect
as part of its `0.1.0` release. See [#272](https://github.com/hyperledger/firefly/issues/272) to track its progress. For now,
you can either deploy Fabconnect yourself or use a cloud provider like [Kaleido](https://www.kaleido.io) which provides
Fabconnect alongside its Fabric peer nodes.

Once you have a Fabconnect instance ready, FireFly then needs three pieces of configuration:

* `config.organizationKey`: the name of the organization's Fabric identity which will be used for signing transactions
* `config.fabconnectUrl`: the HTTP/S URL of the Fabconnect instance FireFly will use
* `config.fabconnectSigner`:  the name of the organization's Fabric identity which will be used for signing transactions.
  See [Identity Management](#identity-management) for how to you can create and enroll the identity using Fabconnect.

These will enable the FireFly deployment to connect to the Fabric blockchain and submit batch pin transactions via
its chaincode on behalf of the organization it's representing.

#### Chaincode

By default, the chart assumes the [FireFly chaincode](../../../smart_contracts/fabric/firefly-go/) is deployed to the
`default-channel` with the name `firefly_go`. If the chaincode was deployed to a different channel or with a different
name you can set `config.fabconnectChannel` and `config.fireflyChaincode` accordingly.

For deploying the chaincode yourself, consult the [Fabric documentation](https://hyperledger-fabric.readthedocs.io/en/latest/deploy_chaincode.html).

#### Identity Management

The Fabric identity FireFly will use for signing transactions on behalf of the organization must be pre-enrolled with
the Fabric CA before deploying FireFly and registration its organization. Fabconnect provides an `/identities` REST API
which makes creating  an identity and enrolling it easy. For example, the following Bash script performs the necessary
API calls to create and enroll an identity named `${ORG_NAME}`:

```shell
identityRegistrationResponse=$(curl --fail -s \
  -X POST \
  -H 'Content-Type: application/json' \
  -d "{ \"name\": \"${ORG_NAME}\", \"type\": \"client\" }" \
  "${FABCONNECT_URL}/identities")

enrollmentSecret=$(echo -n $identityRegistrationResponse | jq -r .secret)
curl --fail -s \
  -X POST \
  -H  'Content-Type: application/json' \
  -d "{ \"secret\": \"${enrollmentSecret}\" }" \
  "${FABCONNECT_URL}/identities/${ORG_NAME}/enroll" | jq -r
```

You can use Bash or whatever scripting / programming language you prefer to enroll the identity. If you wish to enroll
the identity without having to first deploying Fabconnect, please consult the [Fabric CA documentation](https://hyperledger-fabric-ca.readthedocs.io/en/latest/deployguide/use_CA.html).

### Ingress Example

If you have an [`Ingress` controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) deployed
to your cluster, and the chart supports deploying an [`Ingress`](https://kubernetes.io/docs/concepts/services-networking/ingress/)
for the FireFly REST API and websocket subscriptions. For example, if you are using the [`ingress-nginx` controller](https://kubernetes.github.io/ingress-nginx/)
alongside [`cert-manager`](https://cert-manager.io/) you can secure FireFly with TLS and the necessary settings:

```yaml
core:
  ingress:
    enabled: true
    className: nginx # assuming you are using the default ingressClassName for nginx-ingress
    annotations:
      # recommended for handling blob data transfers and broadcasts
      nginx.ingress.kubernetes.io/proxy-body-size: 128m
      # example cert-manager ClusterIssuer for Let's Encrypt
      cert-manager.io/cluster-issuer: letsencrypt-prod
    hosts:
      - host: firefly.acme.org
    tls:
      - secretName: firefly-tls
        hosts:
          - firefly.acme.org
```

### Database Migrations

The database schema FireFly uses for its state must be configured via [migrations](https://www.prisma.io/dataguide/types/relational/what-are-database-migrations).
The chart offers the ability to automatically apply the migrations matching the version of FireFly in use via a `Job`:

```yaml
core:
  jobs:
    postgresMigrations:
      enabled: true
```

The `Job` will be named with the FireFly version in use, and will be automatically replaced and re-run whenever the
version is updated indicating the expected schema could have potentially changed. 

Additionally, FireFly itself can apply its own schema migrations. However, this is not recommended for production use
where an organization could have multiple FireFly nodes sharing the same database:

```yaml
config:
  postgresAutomigrate: true
```

It is recommended to use the migrations `Job` from above in favor of the automatic migrations.

### Auto-Registration

FireFly requires that the organizations within the multi-party system, as well as the individual FireFly
nodes be [registered](https://hyperledger.github.io/firefly/keyconcepts/broadcast.html#firefly-built-in-broadcasts) with
the rest of the network. This can be accomplished using the [FireFly REST API](https://hyperledger.github.io/firefly/swagger/swagger.html#/default/postNewOrganizationSelf),
however the chart offers a registration `Job` which will ensure the organization is registered before then
registering the node:

```yaml
core:
  jobs:
    registration:
      enabled: true
```

### DataExchange HTTPS and cert-manager

The DataExchange HTTPS uses mTLS to securely send messages to other peers. By default, the
chart assumes an mTLS certificate with the proper `subject` and `commonName` is provided
via `dataexchange.tlsSecret.name`.

However, the chart offers the ability to automatically provision and wire up the DataExchange
with an mTLS certificate using [cert-manager](https://cert-manager.io/):

```yaml
dataexchange:
  tlsSecret:
    enabled: false
  
  certificate:
    enabled: true
    issuerRef:
      name: selfsigned-ca
      kind: ClusterIssuer
```

> **NOTE**: the certificate cannot be signed by a self-signed or public CA issuer because cert-manager will not set the
> `subject` and `commonName` properly (see https://github.com/jetstack/cert-manager/issues/3651). We recommend using
> an internal CA issuer instead. An example setup of a CA issuer signed by a self-signed issuer can be found [here](../../manifests/tls-issuers.yaml).

If your DataExchange HTTPS is communicating via `Ingresses`, you will need to enable TLS passthrough
in order for mTLS to work. For example, when using [ingress-nginx](https://kubernetes.github.io/ingress-nginx/) an
annotation can be set on the `Ingress`:

```yaml
  ingress:
    enabled: true
    annotations:
      nginx.ingress.kubernetes.io/ssl-passthrough: "true"
    class: nginx
    hosts:
      - host: firefly-dx.acme.org
```

> **NOTE**: the `tls` section of the `Ingress` does not need to be configured since mTLS is required. Instead,
> it assumes the provided `hosts` must match the `tls[0].hosts` and that the secret is either pre-made or
> provided by cert-manager.

### Tokens via ERC1155 Connector

Chart support for the [ERC1155 token connector](https://github.com/hyperledger/firefly-tokens-erc1155) is coming soon.
See [#272](https://github.com/hyperledger/firefly/issues/272) for updates on its progress.

### Prometheus Support

FireFly comes with an [metrics endpoint](https://prometheus.io/docs/instrumenting/exposition_formats/#text-format-example)
exposed on a separate HTTP server for [Prometheus](https://prometheus.io/) scraping.

By default, the FireFly Prometheus metrics server is enabled. You can turn the server off, or configure its exposed port
and path using the following values:

```yaml
config:
  metricsEnabled: true
  metricsPath: /metrics

core:
  service:
    metricsPort: 5100
```

Additionally, if you are managing Prometheus via the [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator),
you can enable a [`ServiceMonitor`](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/user-guides/getting-started.md#related-resources)
for FireFly with:

```yaml
core:
  metrics:
    serviceMonitor:
      enabled: true
```

## Automated Deployments

Due to Helm's OCI registry support being experimental, below describes how to configure
common deployment automation tooling for consuming the FireFly chart.

### GitOps

#### Flux V2

[Flux V2](https://fluxcd.io/docs/) is a GitOps controller for Kubernetes which currently [does not support Helm OCI registries](https://github.com/fluxcd/source-controller/issues/124).
Instead, one can use a [`GitRepository`](https://fluxcd.io/docs/components/source/gitrepositories/) resource pointed at a specific release tag:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: firefly-helm
spec:
  interval: 10m
  url: "https://github.com/hyperledger/firefly"
  ref:
    tag: helm-v0.1.0
  ignore: |
    /*
    !/deploy/charts/firefly
```

then within a [`HelmRelease`](https://fluxcd.io/docs/components/helm/helmreleases/) resource you can refer to the chart via the `GitRepostiory`:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: firefly
spec:
  chart:
    spec:
      chart: /deploy/charts/firefly
      sourceRef:
        name: firefly-helm
        kind: GitRepository
  interval: 1m
  values: |
    # ...
```

#### ArgoCD

[ArgoCD](https://argo-cd.readthedocs.io/en/stable/) is another GitOps controller for Kubernetes which does support OCI
Helm registries. In order to use the FireFly Helm chart via an ArgoCD [`Application`](https://argo-cd.readthedocs.io/en/stable/user-guide/helm/#declarative),
you must first add the OCI Helm registry for Hyperledger. For example, you can do so using the [CLI](https://argo-cd.readthedocs.io/en/stable/user-guide/commands/argocd_repo_add/):

```shell
argocd repo add ghcr.io/hyperledger/helm --type helm --name hyperledger --enable-oci --username ${USERNAME} --password ${PAT}
```

To declaratively add the registry consult the [documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#repositories).

### Terraform

[Terraform](https://www.terraform.io/) is a CLI tool that enables engineers to "plan" and "apply" infrastructure defined
as code in the [HCL language](https://github.com/hashicorp/hcl). Terraform offers a [Helm provider](https://registry.terraform.io/providers/hashicorp/helm/latest/docs)
for managing Helm releases and their values declaratively. Terraform [does not currently support OCI registries](https://github.com/hashicorp/terraform-provider-helm/issues/633).

As a result, you can configure Terraform to use the FireFly chart by either:

1. Creating a wrapper parent chart with the FireFly chart dependency pre-downloaded and [vendored](https://medium.com/plain-and-simple/dependency-vendoring-dd765be75655).
   See [Using as a Dependency](#using-as-a-dependency) for more information.

2. Pre-downloading the FireFly chart directly using:
    ```shell
    helm pull --version 0.0.1 oci://ghcr.io/hyperledger/helm/firefly
    ```
   then referring to via its filepath location:
    ```hcl
    resource "helm_release" "firefly" {
      name = "firefly"
      chart = "firefly-0.0.1.tgz"
      // ...
    }
    ```