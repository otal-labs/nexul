---
title: Licensing
description: The open-core split, what that means for a contributor, and the CLA.
sidebar:
  order: 8
---

Nexul is one repository under two licenses — open core, not open
source everywhere.

## The split

- Everything **outside `ee/`** is **AGPL-3.0-or-later**. Anyone can run it,
  modify it, and self-host it at any company size, as long as they honor the
  AGPL — including publishing modifications if you offer a modified copy of
  the software over a network.
- Everything **inside `ee/`** is under a commercial license (`ee/LICENSE`).
  That code is public so you can read and patch it, but running it in
  production needs a valid Nexul subscription. Today the directory holds
  only the license; the commercial features are planned, not built.

## Contributing

Outside contributions require a Contributor License Agreement, because the
project has to keep licensing both halves of the codebase. The text is
[`CLA.md`](https://github.com/otal-labs/nexul/blob/master/CLA.md) at the
repository root; you agree to it by ticking the box in the pull request
template. It covers contributions to both halves, including `ee/`.

## Source

This page reflects the [License section of
`README.md`](https://github.com/otal-labs/nexul/blob/master/README.md#license)
and the [`LICENSE`](https://github.com/otal-labs/nexul/blob/master/LICENSE)
file (AGPL-3.0) at the repository root.
