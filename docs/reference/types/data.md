---
layout: default
title: Data
parent: Core Resources
grand_parent: pages.reference
nav_order: 16
---

# Data
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Data

{% include_relative _includes/data_description.md %}

### Example

```json
{
    "id": "4f11e022-01f4-4c3f-909f-5226947d9ef0",
    "validator": "json",
    "namespace": "ns1",
    "hash": "5e2758423c99b799f53d3f04f587f5716c1ff19f1d1a050f40e02ea66860b491",
    "created": "2022-05-16T01:23:15Z",
    "datatype": {
        "name": "widget",
        "version": "v1.2.3"
    },
    "value": {
        "name": "filename.pdf",
        "a": "example",
        "b": {
            "c": 12345
        }
    },
    "blob": {
        "hash": "cef238f7b02803a799f040cdabe285ad5cd6db4a15cb9e2a1000f2860884c7ad",
        "size": 12345,
        "name": "filename.pdf"
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the data resource | [`UUID`](simpletypes#uuid) |
| `validator` | The data validator type | `FFEnum`: |
| `namespace` | The namespace of the data resource | `string` |
| `hash` | The hash of the data resource. Derived from the value and the hash of any binary blob attachment | `Bytes32` |
| `created` | The creation time of the data resource | [`FFTime`](simpletypes#fftime) |
| `datatype` | The optional datatype to use of validation of this data | [`DatatypeRef`](#datatyperef) |
| `value` | The value for the data, stored in the FireFly core database. Can be any JSON type - object, array, string, number or boolean. Can be combined with a binary blob attachment | [`JSONAny`](simpletypes#jsonany) |
| `public` | If the JSON value has been published to shared storage, this field is the id of the data in the shared storage plugin (IPFS hash etc.) | `string` |
| `blob` | An optional hash reference to a binary blob attachment | [`BlobRef`](#blobref) |

## DatatypeRef

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | The name of the datatype | `string` |
| `version` | The version of the datatype. Semantic versioning is encouraged, such as v1.0.1 | `string` |


## BlobRef

| Field Name | Description | Type |
|------------|-------------|------|
| `hash` | The hash of the binary blob data | `Bytes32` |
| `size` | The size of the binary data | `int64` |
| `name` | The name field from the metadata attached to the blob, commonly used as a path/filename, and indexed for search | `string` |
| `public` | If the blob data has been published to shared storage, this field is the id of the data in the shared storage plugin (IPFS hash etc.) | `string` |


