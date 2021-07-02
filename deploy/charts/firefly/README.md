# FireFly

A bare-bones Helm chart for installing a [FireFly](https://github.com/hyperledger-labs/firefly) node with robust templating of its configuration
for development and production scenarios. Additionally, includes FireFly's [default DataExchange](https://github.com/hyperledger-labs/firefly-dataexchange-https) component
for simple, private messaging using HTTPS backed with mTLS.

## TL;DR

```shell
# Deploy a FireFly node w/ some dummy values
$ helm install acme-firefly ./deploy/charts/firefly \
  --set dataexchange.tlsSecret.name=acme-dx-tls \
  --set config.organizationName=acme \
  --set config.organizationIdentity="0xeb7284ce905e0665b7d42cabe31c76c45da1d331" \
  --set config.fireflyContractAddress="0xeb7284ce905e0665b7d42cabe31c76c45da1d254"
```

> **Note**: FireFly requires additional configuration for its blockchain, database, and public storage in order to be fully functional. See [example-values.yaml](ci/it-values.yaml) for an example, and below for more details.

## Introduction



## Prerequisites

* Kubernetes 1.14+
* Helm 3.6.0
* PV provisioner support in the underlying infrastructure

## Installing the Chart

```shell

```

## Uninstalling the Chart

```shell

```

## Parameters

| Parameter                                     | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                | Default                                                       |
|-----------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------|
| `config.debugEnabled` | Enables the FireFly debug port on 6060 and `DEBUG` level logs. | `false` |