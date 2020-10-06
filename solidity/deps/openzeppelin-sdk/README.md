# OpenZeppelin SDK

[![standard-readme compliant](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg)](https://github.com/RichardLitt/standard-readme)
[![CircleCI](https://circleci.com/gh/OpenZeppelin/openzeppelin-sdk/tree/master.svg?style=shield)](https://circleci.com/gh/OpenZeppelin/openzeppelin-sdk/tree/master)
[![lerna](https://img.shields.io/badge/maintained%20with-lerna-cc00ff.svg)](https://lernajs.io/)

> Formerly known as ZeppelinOS

OpenZeppelin is a platform to develop, deploy and operate smart contract
projects on Ethereum and every other EVM and eWASM-powered blockchain.

This repository includes the OpenZeppelin
[Command-Line Interface](https://github.com/OpenZeppelin/openzeppelin-sdk/tree/master/packages/cli#readme) and
[Upgrades Library](https://github.com/OpenZeppelin/openzeppelin-sdk/tree/master/packages/lib#readme).

## Install

First, install [Node.js](http://nodejs.org/) and [npm](https://npmjs.com/).
Then, install the OpenZeppelin SDK running:

```sh
npm install --global @openzeppelin/cli
```

> If you get an `EACCESS permission denied` error while installing, please refer to the [npm documentation on global installs permission errors](https://docs.npmjs.com/resolving-eacces-permissions-errors-when-installing-packages-globally). Alternatively, you may run `sudo npm install --unsafe-perm --global @openzeppelin/cli`, but this is highly discouraged, and you should rather either use a node version manager or manually change npm's default directory.

## Usage

We recommend to use the OpenZeppelin SDK through the `openzeppelin sdk` command-line interface.

To start, create a directory for the project and access it:

```sh
mkdir my-project
cd my-project
```

Use `npm` to create a `package.json` file:

```sh
npm init
```

And initialize the OpenZeppelin SDK project:

```sh
openzeppelin init my-project
```

Now it is possible to use `openzeppelin deploy` to create instances for these contracts that 
later can be upgraded, and many more things.

Run `openzeppelin --help` for more details about thes and all the other functions of the
OpenZeppelin CLI.

The
[OpenZeppelin SDK documentation](https://docs.openzeppelin.com/sdk/2.5)
explains how to build a project using our platform, how to upgrade contracts,
how to share packages for other projects to reuse, how to vouch for the quality
of a package, how to use the JavaScript libraries to operate the project, and
it explains details of the platform and some advanced topics.

## Security

If you find a security issue, please contact us at security@openzeppelin.com. We
give rewards for reported issues, according to impact and severity.

## Maintainers

* [@spalladino](https://github.com/spalladino)
* [@jcarpanelli](https://github.com/jcarpanelli)
* [@ylv-io](https://github.com/ylv-io)

## Community

Join our [Community Forum](https://forum.openzeppelin.com) or
[community channel on Telegram](https://t.me/zeppelinos), where you can talk to
all the OpenZeppelin developers, contributors, partners, and users.

You can also follow the recent developments of the project in the OpenZeppelin [blog](https://blog.openzeppelin.com/) and
[Twitter account](https://twitter.com/openzeppelin).

## Contributing
To set up a local development environment for contributing, clone the repository and run `yarn` in the root of the project. 

Please refer to the [contributing guide](CONTRIBUTING.md) for more details on how to contribute.

## License

[MIT](LICENSE) © OpenZeppelin
