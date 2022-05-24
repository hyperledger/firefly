An `ack` must be sent on a WebSocket for each event delivered to an application.

> Unless `autoack` is set in the [WSStart](./wsstart.html) payload/URL parameters to cause
> automatic acknowledgement.

Your application should specify the `id` of each event that it acknowledges.

If the `id` is omitted, then FireFly will assume the oldest message delivered to the
application that has not been acknowledged is the one the `ack` is associated with.

If multiple subscriptions are started on a WebSocket, then you need to specify the
subscription `namespace`+`name` as part of each `ack`.

If you send an acknowledgement that cannot be correlated, then a [WSError](./wserror.html)
payload will be sent to the application.
