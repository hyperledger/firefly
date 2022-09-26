---
layout: i18n_page
title: pages.release_notes
nav_order: 10
has_children: false
---

# Release Notes

[Full release notes](https://github.com/hyperledger/firefly/releases)

## [Version 1.1 - September 12, 2022](https://github.com/hyperledger/firefly/releases/tag/v1.1.0)

[FireFly 1.1 migration guide](https://github.com/hyperledger/firefly/wiki/Migration-Guide-for-v1.1.0)

What's New:
- Gateway Mode: Connect to many chains with auto-indexing of activities
- Public EVM Chain Support: Manage public chain connections including Ethereum, Polygon, Arbitrum, Binance Smart Chain, Moonbeam, and more.
- Namespaces: Isolated environments within a FireFly runtime allowing independent configuration of plugin and infrastructure components and more
- Connector Toolkit: Quickly build custom connectors
- Pluggable API Security: Plug in your own API security
- Mass Scale Tokens: Support many parallel copies of token plugins for mass scale

## [Version 1.0.3 - July 07, 2022](https://github.com/hyperledger/firefly/releases/tag/v1.0.3)

What's New:
- Adds support for custom URIs for non-fungible tokens and documentation updates
- Deprecate default value for "ffdx"
- Back port of custom URI support for non-fungible tokens
- Update token connector versions
- Back port of "FAQ and FireFly Tutorial updates"

## [Version 1.0.2 - May 12, 2022](https://github.com/hyperledger/firefly/releases/tag/v1.0.2)

What's New:
- Fix invocations on custom Fabric chaincode, which were not properly reporting success/failure status back to FireFly (along with other minor bugfixes).
- De-duplicate existing token approvals in database migration
- Backport docs generation and versioning code for 1.0 stream
- Default fabconnect calls to async
- Set message header type of broadcast/private

## [Version 1.0.1 - May 09, 2022](https://github.com/hyperledger/firefly/releases/tag/v1.0.1)

What's New:
- Fixes for token approvals - previously approvals would intermittently be missed by FireFly or recorded with incorrect details.
- New versions of ERC20/ERC721 connector will assume "no data" support if you create a token pool against an older version of the sample smart contracts.

## [Version 1.0.0 - April 28, 2022](https://github.com/hyperledger/firefly/releases/tag/v1.0.0)

This release includes lots of major hardening, performance improvements, and bug fixes, as well as more complete documentation and OpenAPI specifications.

What's New:
- Massive performance improvements across the board
- Up-to-date documentation and fully annotated OpenAPI specification
- Overhaul of UI
- Cleaner logs and error messages
- Lots of bug fixes and miscellaneous enhancements


## [Version 0.14.0 - March 22, 2022](https://github.com/hyperledger/firefly/releases/tag/v0.14.0)

What's New:
- Major UI updates including Activity, Blockchain, Off-Chain, Tokens, Network Map, and My Node sections
- Custom contract APIs
- Enhanced subscription filters
- Event API enrichment
- Performance updates
- Bug fixes

## [Version 0.13.0 - February 14, 2022](https://github.com/hyperledger/firefly/releases/tag/v0.13.0)

What's New:
- Hardening release with significant rework to core of FireFly, mostly to fix issues exposed by the performance testing.
- Support for running on ARM-based M1 processors
- Rewrite of the message batching and event aggregation logic inside FireFly, to fix numerous edge cases with lost or hung messages
- Hardening of operations and transactions to behave more consistently across all types
- Metrics reporting to Prometheus
- Continued development to support custom on-chain logic (still in preview)

## [Version 0.12.0 - February 02, 2022](https://github.com/hyperledger/firefly/releases/tag/v0.12.0)

What's New:
- All APIs deprecated in v0.11.0 or earlier are removed
- Preview of custom on-chain logic
- Support for new ERC20 / ERC721 connector
- Overhaul of Transaction type and new BlockchainEvent type
- Support for delivery confirmations via DX plugin

## [Version 0.11.0 - November 22, 2021](https://github.com/hyperledger/firefly/releases/tag/v0.11.0)

What's New:
- Significant hardening and enhanced token functionality
- Major web UI overhaul
- Optimized database operations for increased transactional throughput
- Fixed PostgreSQL database migrations