# Mattermost Yandex Calendar Plugin

[![Build Status](https://img.shields.io/circleci/project/github/danilvalov/mattermost-plugin-yandex-calendar/master.svg)](https://circleci.com/gh/danilvalov/mattermost-plugin-yandex-calendar)
[![Code Coverage](https://img.shields.io/codecov/c/github/danilvalov/mattermost-plugin-yandex-calendar/master.svg)](https://codecov.io/gh/danilvalov/mattermost-plugin-yandex-calendar)
[![Release](https://img.shields.io/github/v/release/danilvalov/mattermost-plugin-yandex-calendar)](https://github.com/danilvalov/mattermost-plugin-yandex-calendar/releases/latest)
[![HW](https://img.shields.io/github/issues/danilvalov/mattermost-plugin-yandex-calendar/Up%20For%20Grabs?color=dark%20green&label=Help%20Wanted)](https://github.com/danilvalov/mattermost-plugin-yandex-calendar/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Up+For+Grabs%22+label%3A%22Help+Wanted%22)

A [Mattermost](https://mattermost.com) calendar plugin that syncs **Yandex Calendar** over **CalDAV** using a Yandex **app password** (no OAuth).

## Documentation

[About](docs/about.md) | [Set up](docs/setup.md) | [Configure](docs/configuration.md) | [Use](docs/usage.md) | [CalDAV notes](docs/caldav-yandex.md)

## Features

- Daily summary, reminders, status sync, and slash-command agendas
- Connect with Yandex email + app password through `/ycal connect` and the CalDAV connect page
- Background polling for event changes (Yandex does not support calendar push webhooks)

## Build

Requirements: Go 1.24+, Node/npm for the webapp bundle.

```bash
git submodule update --init --recursive   # once after clone; `make dist` also initializes submodules
make verify-submodule-pins   # optional: confirm go-webdav gitlink matches upstream v0.7.0 (needs network)
make dist
```

Install the generated tarball under **System Console > Plugins**.

## Repository layout

- `ycal/` — Yandex CalDAV provider (`remote.Client` implementation)
- `calendar/` — plugin server logic (commands, jobs, engine, API)
- `third_party/` — Git submodule [`emersion/go-webdav`](https://github.com/emersion/go-webdav) (v0.7.0 + Yandex ETag patch). Patches live under `patches/` and are applied by `make submodules`.

## Development

Read Mattermost documentation about the [Developer Workflow](https://developers.mattermost.com/integrate/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/integrate/plugins/developer-setup/) for more information about developing and extending plugins.

## How to release

To trigger a release, run one of the following from a protected branch (see `Makefile`):

- `make patch`, `make minor`, or `make major` for a release
- `make patch-rc`, `make minor-rc`, or `make major-rc` for a release candidate
