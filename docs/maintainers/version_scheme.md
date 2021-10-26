---
layout: default
title: Versioning Scheme
parent: Maintainers
nav_order: 1
---

# Versioning Scheme

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

This page describes FireFly's versioning scheme

---

## Semantic versioning
FireFly follows [semantic versioning](https://semver.org/). In summary, this means:

Given a version number MAJOR.MINOR.PATCH, increment the:

- MAJOR version when you make incompatible API changes,
- MINOR version when you add functionality in a backwards compatible manner, and
- PATCH version when you make backwards compatible bug fixes.
- Additional labels for pre-release and build metadata are available as extensions to the MAJOR.MINOR.PATCH format.

When creating a new release, the release name and tag should be the semantic version should be prefixed with a `v` . For example, a certain release name/tag could be `v0.9.0`.

## Pre-release test versions
For pre-release versions for testing, we append a date and index to the end of the most recently released version. For example, if we needed to create a pre-release based on `v0.9.0` and today's date is October 22, 2021, the version name/tag would be: `v0.9.0-20211022-01`. If for some reason you needed to create another pre-release version in the same day (hey, stuff happens), the name/tag for that one would be `v0.9.0-20211022-02`.

## Candidate releases
For pre-releases that are candidates to become a new major or minor release, the release name/tag will be based on the release that the candidate *will become* (as opposed to the test releases above, which are based on the previous release). For example, if the current latest release is `v0.9.0` but we want to create an alpha release for 1.0, the release name/tag would be `v1.0.0-alpha-01`.