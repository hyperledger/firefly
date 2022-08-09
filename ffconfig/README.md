# FireFly configuration tool

A tool for managing and migrating config files for Hyperledger FireFly.

## Installation

If you have a local Go development environment, and you have included `${GOPATH}/bin` in your path, you can install with:

```sh
go install github.com/hyperledger/firefly/ffconfig@latest
```

## Usage

### Migration

Migrating a config file will remove all deprecated config for a particular version of FireFly, and replace it with
the latest correct syntax.

```
ffconfig migrate -f firefly.core.yml
```

You may optionally specify `--from` and `--to` versions to run a subset of the migrations, and may specify `-o` or
simply redirect stdout to write the results to a file.

View the source code for all current migrations at [migrate/migrations.go](migrate/migrations.go).
