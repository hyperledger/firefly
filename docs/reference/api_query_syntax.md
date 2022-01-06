---
layout: default
title: API Query Syntax
parent: Reference
nav_order: 1
---

# API Query Syntax
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Syntax Overview

REST collections provide filter, `skip`, `limit` and `sort` support.
- The field in the message is used as the query parameter
  - Syntax: `field=[modifiers][operator]match-string`
- When multiple query parameters are supplied these are combined with AND
- When the same query parameter is supplied multiple times, these are combined with OR

## Example API Call

`GET` `/api/v1/messages?confirmed=>0&type=broadcast&topic=t1&topic=t2&context=@someprefix&sort=sequence&descending&skip=100&limit=50`

This states:

- Filter on `confirmed` greater than 0
- Filter on `type` exactly equal to `broadcast`
- Filter on `topic` exactly equal to `t1` _or_ `t2`
- Filter on `context` containing the case-sensitive string `someprefix`
- Sort on `sequence` in `descending` order
- Paginate with `limit` of `50` and `skip` of `100` (e.g. get page 3, with 50/page)

Table of filter operations, which must be the first character of the query string (after the `=` in the above URL path example)

### Operators

Operators are a type of comparison operation to
perform against the match string.

| Operator | Description                        |
|----------|------------------------------------|
| `=`      | Equal                              |
| (none)   | Equal (shortcut)                   |
| `@`      | Containing                         |
| `^`      | Starts with                        |
| `$`      | Ends with                          |
| `<<`     | Less than                          |
| `<`      | Less than (shortcut)               |
| `<=`     | Less than or equal                 |
| `>>`     | Greater than                       |
| `>`      | Greater than (shortcut)            |
| `>=`     | Greater than or equal              |

> Shortcuts are only safe to use when your match
> string starts with `a-z`, `A-Z`, `0-9`, `-` or `_`.

### Modifiers

Modifiers can appear before the operator, to change its
behavior.

| Modifier | Description                        |
|----------|------------------------------------|
| `!`      | Not - negates the match            |
| `:`      | Case insensitive                   |

> Characters `=`,`@`,`$`,`!` and `:` should technically be encoded
> in URLs, but in practice should function fine without encoding.

## Detailed examples

| Example      | Description                                |
|--------------|--------------------------------------------|
| `cat`        | Equals "cat"                               |
| `=cat`       | Equals "cat" (same)                        |
| `!=cat`      | Not equal to "cat"                         |
| `:=cat`      | Equal to "CAT", "cat", "CaT etc.           |
| `!:cat`      | Not equal to "CAT", "cat", "CaT etc.       |
| `=!cat`      | Equal to "!cat" (! is after operator)      |
| `^cats/`     | Starts with "cats/"                        |
| `$_cat`      | Ends with with "_cat"                      |
| `!:^cats/`   | Does not start with "cats/", "CATs/" etc.  |
| `!$-cat`     | Does not end with "-cat"                   |
