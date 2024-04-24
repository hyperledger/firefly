# Welcome to the documentation template

This repository serves as a template for creating documentation for Hyperledger projects. The template utilizes MkDocs (documentation at [mkdocs.org](https://www.mkdocs.org)) and the theme Material for MkDocs (documentation at [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/)). Material adds a number of extra features to MkDocs, and Hyperledger repositories can take advantage of the theme's [Insiders](https://squidfunk.github.io/mkdocs-material/insiders/) capabilities.

[Material for MkDocs]: https://squidfunk.github.io/mkdocs-material/
[Mike]: https://github.com/jimporter/mike

## Prerequisites

To test the documents and update the published site, the following tools are needed:

- A Bash shell
- git
- Python 3
- The [Material for Mkdocs] theme.
- The [Mike] MkDocs plugin for publishing versions to gh-pages.
  - Not used locally, but referenced in the `mkdocs.yml` file and needed for
    deploying the site to gh-pages.

### git
`git` can be installed locally, as described in the [Install Git Guide from GitHub](https://github.com/git-guides/install-git).

### Python 3
`Python 3` can be installed locally, as described in the [Python Getting Started guide](https://www.python.org/about/gettingstarted/).

### Mkdocs

The Mkdocs-related items can be installed locally, as described in the [Material
for Mkdocs] installation instructions. The short, case-specific version of those
instructions follow:

```bash
pip install -r requirements.txt
```

### Verify Setup

To verify your setup, check that you can run `mkdocs` by running the command `mkdocs --help` to see the help text.

## Useful MkDocs Commands

The commands you will usually use with `mkdocs` are:

* `mkdocs serve` - Start the live-reloading docs server.
* `mkdocs build` - Build the documentation site.
* `mkdocs -h` - Print help message and exit.

## Adding Content

The basic process for adding content to the site is:

- Create a new markdown file under the `docs` folder
- Add the new file to the table of contents (`nav` section in the `mkdocs.yml` file)

If you are using this as a template for creating your own documentation, please see [the instructions for customization](./docs/index.md).

## Repository layout

    mkdocs.yml    # The configuration file.
    docs/
        index.md  # The documentation homepage.
        ...       # Other markdown pages, images and other files.
