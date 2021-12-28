# FireFly

[Hyperledger FireFly](https://hyperledger.github.io/firefly/) is an implementation of a [multi-party system](https://github.com/hyperledger/firefly#multi-party-systems) that
simplifies data orchestration on top of blockchain and other peer-to-peer technologies.

This chart bootstraps a FireFly deployment on a [Kubernetes](https://kubernetes.io/) cluster using the [Helm](https://helm.sh/) package manager.

### Table of Contents

* [Prerequisites](#prerequisites)
* [Get Repo Info](#get-repo-info)
* [Install Chart](#install-chart)
* [Uninstall Chart](#uninstall-chart)
* [Upgrading Chart](#upgrading-chart)
* [Using as a Dependency](#using-as-a-dependency)
* [Deployment Architecture](#deployment-architecture)
    * [Infrastructural Dependecies](#infrastructural-dependecies)
* [Configuration](#configuration)
    * [Configuration File Templating](#configuration-file-templating)
        * [Ethereum](#ethereum)
        * [Fabric](#fabric)
    * [Ingress Examples](#ingress-examples)
    * [SQL Database Migrations](#sql-database-migrations)
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
the FireFly chart, as a result one must login to the [GHCR](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
to download the chart:

```shell
export HELM_EXPERIMENTAL_OCI=1

helm registry login ghcr.io
```

> **NOTE**: it is recommended to use a [GitHub personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
> when authenticating to the GHCR registry as opposed to using your GitHub password.

## Install Chart

```shell
helm install firefly --version 0.1.0 oci://ghcr.io/hyperledger/helm/firefly
```

_See [configuration](#Configuration) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

## Uninstall Chart

```shell
helm uninstall firefly
```

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Chart

```shell
helm upgrade firefly --version 0.1.1 oci://ghcr.io/hyperledger/helm/firefly
```

_See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) for command documentation._

## Using as a Dependency

You can also use the FireFly chart within your own parent chart's `Chart.yaml`:

```yaml
dependencies:
  # ...
  - name: firefly
    repository: "oci://ghcr.io/hyperledger/helm/"
    version: 0.1.0
```

Then download the chart dependency into your parent chart:

```shell
helm dep up path/to/parent-chart
```

_See [helm dependency](https://helm.sh/docs/helm/helm_dependency/) for command documentation._


## Deployment Architecture

However, using blockchain directly can be challenging, and in multi-party systems there is typically
off-chain data orchestration (i.e. private and blob  data exchange) that is required as well.
FireFly provides a REST API with an event-driven paradigm that makes building such multi-party interactions via
decentralized application much simpler, while providing extensible connector plugins that enable swapping out the
underlying blockchain and off-chain infrastructure easily.

<img src="./../../../images/helm_chart_deployment_architecture.jpg" />

### Infrastructural Dependecies

Based on the above, FireFly has several infrastructural dependencies:

* Blockchain connector (Fabconnect -> Fabric, Ethconnect -> Ethereum)
* An Ethereum smart contract or Fabric chaincode deployed to the underlying blockchain
* Private data exchange (HTTPS + mTLS, Kafka + mTLS, etc.)
* Database (PostgreSQL, etc.)
* Shared storage (IPFS)

## Configuration

### Configuration File Templating



#### Ethereum

see eth-values.yaml for an example

#### Fabric

see fab-values.yaml for an example

### Ingress Examples



```yaml
core:
  ingress:
    enabled: true
    className: nginx
    annotations:
      # recommended for handling blob data transfers
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

### SQL Database Migrations

```yaml
config:
  postgresMigrationJob: true
```

### Auto-Registration

```yaml
config:
  registrationJob: true
```

### DataExchange HTTPS and cert-manager

[cert-manager]() ...

mention https://github.com/jetstack/cert-manager/issues/3651

```yaml
dataexchange:
  certificate:
    enabled: true
    issuerRef:
      name: selfsigned-ca
      kind: ClusterIssuer
```

if your DataExchange HTTPS is communicating via `Ingresses`, you will need to enable TLS passthrough
in order for mTLS to work. For example, when using [nginx-ingress]() an annotation can be set on the
`Ingress`:

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
> sprovided by cert-manager.



### Tokens via ERC1155 Connector

Chart support for the [ERC1155 token connector](https://github.com/hyperledger/firefly-tokens-erc1155) is coming soon.
See [#218](https://github.com/hyperledger/firefly/issues/218) and [#272](https://github.com/hyperledger/firefly/issues/272)
for updates on its progress.

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
you can enable a [`ServiceMonitor`]() for FireFly with:

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

[ArgoCD](https://argo-cd.readthedocs.io/en/stable/)

via the [CLI](https://argo-cd.readthedocs.io/en/stable/user-guide/commands/argocd_repo_add/):
```shell
argocd repo add ghcr.io/hyperledger/helm --type helm --name hyperledger --enable-oci --username ${USERNAME} --password ${PAT}
```

https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#repositories

```yaml
apiVersion:
kind: Application
```

### Terraform

```shell
helm pull --version 0.1.0 oci://ghcr.io/hyperledger/helm/firefly
```