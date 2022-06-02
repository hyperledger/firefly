Data is a uniquely identified piece of data available for retrieval or transfer.

Multiple data items can be attached to a message when sending data off-chain
to another party in a multi-party system. Note that if you pass data in-line when
sending a message, those data elements will be stored separately to the message
and available to retrieve separately later.

An UUID is allocated to each data resource.

A hash is also calculated as follows:

- If there is only data, the hash is of the `value` serialized as JSON with
  no additional whitespace (order of the keys is retained from the original
  upload order).
- If there is only a `blob` attachment, the hash is of the blob data.
- There is is both a `blob` and a `value`, then the hash is a hash of the
  concatenation of a hash of the value and a hash of the blob.

### Value - JSON data stored in the core database

Each data resource can contain a `value`, which is any JSON type. String, number,
boolean, array or object. This value is stored directly in the FireFly database.

If the value you are storing is not JSON data, but is small enough you want it to
be stored in the core database, then use a JSON string to store an encoded form
of your data (such as XML, CSV etc.).

### Datatype - validation of agreed data types

A datatype can be associated with your data, causing FireFly to verify the
`value` against a schema before accepting it (on upload, or receipt from another
party in the network).

These datatypes are pre-established via broadcast messages, and support versioning.
Use this system to enforce a set of common data types for exchange of data
across your business network, and reduce the overhead of data verification\
required in the application/integration tier.

> More information in the [Datatype](./datatype.html) section

### Blob - binary data stored via the Data Exchange

Data resources can also contain a `blob` attachment, which is stored via the
Data Exchange plugin outside of the FireFly core database. This is intended for
large data payloads, which might be structured or unstructured. PDF documents,
multi-MB XML payloads, CSV data exports, JPEG images video files etc.

A Data resource can contain both a `value` JSON payload, and a `blob` attachment,
meaning that you bind a set of metadata to a binary payload. For example
a set of extracted metadata from OCR processing of a PDF document.

One special case is a filename for a document. This pattern
is so common for file/document management scenarios, that special handling
is provided for it.  If a JSON object is stored in `value`, and it has a property
called `name`, then this value forms part of the data hash (as does every field
in the `value`) and is stored in a separately indexed `blob.name` field.

The upload REST API provides an `autometa` form field, which can be set to ask
FireFly core to automatically set the `value` to contain the filename, size, and
MIME type from the file upload.

