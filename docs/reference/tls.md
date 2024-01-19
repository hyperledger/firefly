---
layout: i18n_page
title: pages.tls
parent: pages.reference
nav_order: 11
---

# TLS
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## TLS Overview


To enable TLS in Firefly, there is a configuration available to provide certificates and keys.

The common configuration is as such:

```yaml
tls:
    enabled: true/false # Toggle on or off TLS
    caFile: <path to the CA file you want the client or server to trust>
    certFile: <path to the cert file you want the client or server to use when performing authentication in mTLS>
    keyFile: <path to the priavte key file you want the client or server to use when performing  authentication in mTLS>
    clientAuth: true/false # Only applicable to the server side, to toggle on or off client authentication
    requiredDNAttributes: A set of required subject DN attributes. Each entry is a regular expression, and the subject certificate must have a matching attribute of the specified type (CN, C, O, OU, ST, L, STREET, POSTALCODE, SERIALNUMBER are valid attributes)	
```

**NOTE** The CAs, certificates and keys have to be in PEM format. 

## Configuring TLS for the API server

Using the above configuration, we can place it under the `http` config and enable TLS or mTLS for any API call.

[See this config section for details](config.html#httptls)

## Configuring TLS for the webhooks

Using the above configuration, we can place it under the `events.webhooks` config and enable TLS or mTLS for any webhook call.

[See this config section for details](config.html#eventswebhookstls)


## Configuring clients and websockets

Firefly has a set of HTTP clients and websockets that communicate the external endpoints and services that could be secured using TLS. 
In order to configure these clients, we can use the same configuration as above in the respective places in the config which relate to those clients. 

For example, if you wish to configure the ethereum blockchain connector with TLS you would look at [this config section](config.html#pluginsblockchainethereumethconnecttls)

For more clients, search in the [configuration reference](config.html) for a TLS section.


## Enhancing validation of certificates

In the case where we want to verify that a specific client certificate has certain attributes we can use the `requiredDNAtributes` configuration as described above. This will allow you by the means of a regex expresssion matching against well known distinguished names (DN). To learn more about a DNs look at [this document](https://datatracker.ietf.org/doc/rfc4514/)
