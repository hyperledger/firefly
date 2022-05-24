---
layout: default
title: Configuration Reference
parent: Reference
nav_order: 3
---

# Configuration Reference
{: .no_toc }

<!-- ## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc} -->

---


## log

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|compress|Determines if the rotated log files should be compressed using gzip|`boolean`|`<nil>`
|filename|Filename is the file to write logs to.  Backup log files will be retained in the same directory|`string`|`<nil>`
|filesize|MaxSize is the maximum size the log file before it gets rotated|[`BytesSize`](https://pkg.go.dev/github.com/docker/go-units#BytesSize)|`<nil>`
|forceColor|Force color to be enabled, even when a non-TTY output is detected|`boolean`|`<nil>`
|includeCodeInfo|Enables the report caller for including the calling file and line number, and the calling function. If using text logs, it uses the logrus text format rather than the default prefix format.|`boolean`|`<nil>`
|level|The log level - error, warn, info, debug, trace|`string`|`<nil>`
|maxAge|The maximum time to retain old log files based on the timestamp encoded in their filename.|[`time.Duration`](https://pkg.go.dev/time#Duration)|`<nil>`
|maxBackups|Maximum number of old log files to retain|`int`|`<nil>`
|noColor|Force color to be disabled, event when TTY output is detected|`boolean`|`<nil>`
|timeFormat|Custom time format for logs|[Time format](https://pkg.go.dev/time#pkg-constants) `string`|`<nil>`
|utc|Use UTC timestamps for logs|`boolean`|`<nil>`

## log.json

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|enabled|Enables JSON formatted logs rather than text. All log color settings are ignored when enabled.|`boolean`|`<nil>`

## log.json.fields

|Key|Description|Type|Default Value|
|---|-----------|----|-------------|
|file|configures the JSON key containing the calling file|`string`|`<nil>`
|func|Configures the JSON key containing the calling function|`string`|`<nil>`
|level|Configures the JSON key containing the log level|`string`|`<nil>`
|message|Configures the JSON key containing the log message|`string`|`<nil>`
|timestamp|Configures the JSON key containing the timestamp of the log|`string`|`<nil>`