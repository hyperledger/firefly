---
layout: default
title: API POST Syntax
parent: Reference
nav_order: 2
---

# API POST Syntax
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Syntax Overview

Endpoints that allow submitting a transaction allow an optional query parameter called `confirm`. When `confirm=true` is set in the query string, FireFly will wait to send an HTTP response until the message has been confirmed. This means, where a blockchain transaction is involved, the HTTP request will not return until the blockchain transaction is complete.

This is useful for endpoints such as registration, where the client app cannot proceed until the transaction is complete and the member/node is registered. Rather than making a request to register a member/node and then repeatedly polling the API to check to see if it succeeded, an HTTP client can use this query parameter and block until registration is complete.

> **NOTE**: This does *not* mean that any other member of the network has received, processed, or responded to the message. It just means that the transaction is complete from the perspective of the FireFly node to which the transaction was submitted.

## Example API Call

`POST` `/api/v1/messages/broadcast?confirm=true`

This will broadcast a message and wait for the message to be confirmed before returning.