A datatype defines the format of some data that can be shared between parties, in a way
that FireFly can enforce consistency of that data against the schema.

Data that does not match the schema associated with it will not be accepted on
upload to FireFly, and if this were bypassed by a participant in some way
it would be rejected by all parties and result in a `message_rejected` event
(rather than `message_confirmed` event).

Currently JSON Schema validation of data is supported.

The system for defining datatypes is pluggable, to support other schemes in the future,
such as XML Schema, or CSV, EDI etc.
