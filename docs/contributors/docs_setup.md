---
layout: default
title: Contributing to Documentation
parent: Contributors
nav_order: 5
---

# Contributing to Documentation

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

This guide will walk you through setting up your machine for contributing to FireFly documentation

---

## Dependencies

You will need a few prerequisites set up on your machine before you can build FireFly from source. We recommend doing development on macOS, Linux, or WSL 2.0.

### macOS
macOS already comes with `ruby` and `bundle` installed and set up already. You don't need to do anything extra.

### Linux
You will need to install `ruby` and `bundle` and have them on your path to build and serve the docs locally. On Ubuntu, to install dependencies, run the following commands:

```
sudo apt-get update
sudo apt-get install ruby ruby-bundler
```

## Build and serve the docs locally
To build and serve the docs locally, from the project root, navigate to the docs directory:

```
cd docs
```

And start the Jekyll test server: 

```
bundle exec jekyll serve
```

You should now be able to open [http://localhost:4000](http://localhost:4000) in your browser and see a locally hosted version of the doc site.

As you make changes to files in the `./docs` directory, Jekyll will automatically rebuild the pages, and notify you of any errors or warnings in your terminal.

> **NOTE**: If you make changes to any page, be sure save the file and refresh your browser to see the change. Jekyll doesn't hot reload the browser for you automatically.