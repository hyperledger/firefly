---
layout: default
title: Setting up a local development environment
parent: Contributors
nav_order: 1
---


# Setting up a local development environment

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

This guide will walk you through setting up your machine for contributing to FireFly, specifically the FireFly core.

---

## Dependencies

You will need a few prerequisites set up on your machine before you can build FireFly from source. We recommend doing development on macOS, Linux, or WSL 2.0.

- Go (1.16 or newer) 
- make
- GCC (?)
- There are probably more that I'm forgetting...

The [FireFly CLI](https://github.com/hyperledger/firefly-cli) is the recommended path for running a local development stack. It has its [own set of prerequisites](https://github.com/hyperledger/firefly-cli#prerequisites) as well.

## Building FireFly

After installing dependencies, building FireFly from source is very easy. Just clone the repo:

```
git clone git@github.com:hyperledger/firefly.git && cd firefly
```

And run the `Makefile` to run tests, and compile the app

```
make
```

If you want to install the binary on your path (assuming your Go Home is already on your path), from inside the project directory you can simply run:

```
go install
```


## Install the CLI

Please check the CLI Installation instructions for the best way to install the CLI on your machine:
https://github.com/hyperledger/firefly-cli#install-the-cli

## Set up a development stack

Now that you have both FireFly and the FireFly CLI installed, it's time to create a development stack. The CLI can be used to create a docker-compose environment that runs the entirety of a FireFly network. This will include several different processes for each member of the network. This is very useful for people that want to build apps that use FireFly's API. It can also be useful if you want to make changes to FireFly itself, however we need to set up the stack slightly differently in that case.

Essentially what we are going to do is have docker-compose run everything in the FireFly network _except_ one FireFly core process. We'll run this FireFly core process on our host machine, and configure it to connect to the rest of the microservices running in docker-compose. This means we could launch FireFly from Visual Studio Code or some other IDE and use a debugger to see what's going on inside FireFly as it's running.

To do this, we're going to add `--external 1` to the end of our command to create the new stack:

```
ff init --external 1
```

This tells the CLI that we want to manage one of the FireFly core processes outside the docker-compose stack. For convenience, the CLI will still generate a config file for this process though. At a certain point in the startup process, the CLI will pause and wait for up to two minutes for you to start the other FireFly node. It will also print out the command line which can be copied and pasted into another terminal window to run FireFly.

> **NOTE**: The first time you run FireFly with a fresh database, it will need a directory of database migrations to apply to the empty database. If you run FireFly from the `firefly` project directory you cloned from GitHub, it will automatically find these and apply them. If you run it from somewhere else, you will have to point FireFly to the migrations on your own.

Now you should have a full FireFly stack up and running, and be able to debug FireFly using your IDE. Happy hacking!