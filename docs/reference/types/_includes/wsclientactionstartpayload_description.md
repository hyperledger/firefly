The `start` payload is sent after an application connects to a WebSocket, to start
delivery of events over that connection.

The start command can refer to a subscription by name in order to reliably receive all matching
events for that subscription, including those that were emitted when the application
was disconnected.

Alternatively the start command can request `"ephemeral": true` in order to dynamically create a new
subscription that lasts only for the duration that the connection is active.
