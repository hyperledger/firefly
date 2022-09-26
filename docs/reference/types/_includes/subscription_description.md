Each [Subscription](subscription.html#subscription) tracks delivery of events to a particular
application, and allows FireFly to ensure that messages are delivered reliably
to that application.

[![FireFly Event Subscription Model](../../images/firefly_event_subscription_model.jpg)](../../images/firefly_event_subscription_model.jpg)

### Creating a subscription

Before you can connect to a subscription, you must create it via the REST API.

> One special case where you do not need to do this, is Ephemeral WebSocket
> connections (described below).
> For these you can just connect and immediately start receiving events.

When creating a new subscription, you give it a `name` which is how you will
refer to it when you connect.

You are also able to specify server-side filtering that should be performed
against the event stream, to limit the set of events that are sent to your
application.

All subscriptions are created within a `namespace`, and automatically filter
events to only those emitted within that namespace.

You can create multiple subscriptions for your application, to request
different sets of server-side filtering for events. You can then request
FireFly to deliver events for both subscriptions over the same WebSocket
(if you are using the WebSocket transport). However, delivery order is
not assured between two subscriptions.

### Subscriptions and workload balancing

You can have multiple scaled runtime instances of a single application,
all running in parallel. These instances of the application all share a
single subscription.

> Each event is only delivered once to the subscription, regardless of how
> many instances of your application connect to FireFly.

With multiple WebSocket connections active on a single subscription,
each event might be delivered to different instance of your application.
This means workload is balanced across your instances. However, each
event still needs to be acknowledged, so delivery processing order
can still be maintained within your application database state.

If you have multiple different applications all needing their own copy of
the same event, then you need to configure a separate subscription
for each application.

### Pluggable Transports

Hyperledger FireFly has two built-in transports for delivery of events
to applications - WebSockets and Webhooks.

The event interface is fully pluggable, so you can extend connectivity
over an external event bus - such as NATS, Apache Kafka, Rabbit MQ, Redis etc.

### WebSockets

If your application has a back-end server runtime, then WebSockets are
the most popular option for listening to events. WebSockets are well supported
by all popular application development frameworks, and are very firewall friendly
for connecting applications into your FireFly server.

> Check out the [@hyperledger/firefly-sdk](https://www.npmjs.com/package/@hyperledger/firefly-sdk)
> SDK for Node.js applications, and the [hyperledger/firefly-common](https://github.com/hyperledger/firefly-common)
> module for Golang applications. These both contain reliable WebSocket clients for your event listeners.
>
> A Java SDK is a roadmap item for the community.

#### WebSocket protocol

FireFly has a simple protocol on top of WebSockets:

1. Each time you connect/reconnect you need to tell FireFly to start
   sending you events on a particular subscription. You can do this in two
   ways (described in detail below):
    1. Send a [WSStart](./wsstart.html) JSON payload
    2. Include a `namespace` and `name` query parameter in the URL when you
       connect, along with query params for other fields of [WSStart](./wsstart.html)
2. One you have started your subscription, each event flows from
   the server, to your application as a JSON [Event](./event.html) payload
3. For each event you receive, you need to send a [WSAck](./wsack.html) payload.
    - Unless you specified `autoack` in step (1)

> The SDK libraries for FireFly help you ensure you send the `start`
> payload each time your WebSocket reconnects.

#### Using `start` and `ack` explicitly

Here's an example [websocat](https://github.com/vi/websocat) command
showing an explicit `start` and `ack`.

```sh
$ websocat ws://localhost:5000/ws
{"type":"start","namespace":"default","name":"docexample"}
# ... for each event that arrives here, you send an ack ...
{"type":"ack","id":"70ed4411-57cf-4ba1-bedb-fe3b4b5fd6b6"}
```

When creating your subscription, you can set `readahead` in order to
ask FireFly to stream a number of messages to your application,
ahead of receiving the acknowledgements.

> `readahead` can be a powerful tool to increase performance, but does
> require your application to ensure it processes events in the correct
> order and sends exactly one `ack` for each event.

#### Auto-starting via URL query and `autoack`

Here's an example [websocat](https://github.com/vi/websocat) where we use
URL query parameters to avoid the need to send a `start` JSON payload.

We also use `autoack` so that events just keep flowing from the server.

```sh
$ websocat "ws://localhost:5000/ws?namespace=default&name=docexample&autoack"
# ... events just keep arriving here, as the server-side auto-acknowledges
#     the events as it delivers them to you.
```

> Note using `autoack` means you can _miss events_ in the case of a disconnection,
> so should not be used for production applications that require at-least-once delivery.

#### Ephemeral WebSocket subscriptions

FireFly WebSockets provide a special option to create a subscription dynamically, that
only lasts for as long as you are connected to the server.

We call these `ephemeral` subscriptions.

Here's an example [websocat](https://github.com/vi/websocat) command
showing an an ephemeral subscription - notice we don't specify a `name` for the
subscription, and there is no need to have already created the subscription
beforehand.

Here we also include an extra query parameter to set a server-side filter, to only
include message events.

```sh
$ websocat "ws://localhost:5000/ws?namespace=default&ephemeral&autoack&filter.events=message_.*"
{"type":"start","namespace":"default","name":"docexample"}
# ... for each event that arrives here, you send an ack ...
{"type":"ack","id":"70ed4411-57cf-4ba1-bedb-fe3b4b5fd6b6"}
```

> Ephemeral subscriptions are very convenient for experimentation, debugging and monitoring.
> However, they do not give reliable delivery because you only receive events that
> occur while you are connected. If you disconnect and reconnect, you will miss all events
> that happened while your application was not listening.

### Webhooks

The Webhook transport allows FireFly to make HTTP calls against your application's API
when events matching your subscription are emitted.

This means the direction of network connection is from the FireFly server, to the
application (the reverse of WebHooks). Conversely it means you don't need to add
any connection management code to your application - just expose and API that
FireFly can call to process the events.

> Webhooks are great for serverless functions (AWS Lambda etc.), integrations
> with SaaS applications, and calling existing APIs.

The FireFly configuration options for a Webhook subscription are very flexible,
allowing you to customize your HTTP requests as follows:

- Set the HTTP request details:
  - Method, URL, query, headers and input body
- Wait for a invocation of the back-end service, before acknowledging
  - To retry requests to your Webhook on a non-`2xx` HTTP status code
    or other error, then you should enable and configure
    [events.webhooks.retry](../../config.html#eventswebhooksretry)
  - The event is acknowledged once the request (with any retries), is
    completed - regardless of whether the outcome was a success or failure.
- Use `fastack` to acknowledge against FireFly immediately and make multiple
  parallel calls to the HTTP API in a fire-and-forget fashion.
- Set the HTTP request details dynamically from `message_confirmed` events:
  - Map data out of the first `data` element in message events
  - Requires `withData` to be set on the subscription, in addition to the
    `input.*` configuration options
- Can automatically generate a "reply" message for `message_confirmed` events:
  - Maps the response body of the HTTP call to data in the reply message
  - Sets the `cid` and `topic` in the reply message to match the request
  - Sets a `tag` in the reply message, per the configuration, or dynamically
    based on a field in the input request data.


