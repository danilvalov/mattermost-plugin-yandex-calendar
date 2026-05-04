# Mattermost Yandex Calendar integration

This plugin integrates Mattermost with **Yandex Calendar** using the CalDAV protocol. Users link their account with an **app password** from Yandex ID; the server stores credentials encrypted in the plugin KV store.

Slash commands, reminders, and the settings panel follow the same calendar UX patterns as Mattermost’s official calendar plugins; this project adds a `ycal` remote that talks to Yandex over CalDAV and includes server-side connect and polling where push webhooks are not available.

- [Set up](setup.md)
- [Configure](configuration.md)
- [Use](usage.md)
- [Yandex CalDAV notes](caldav-yandex.md) (technical)
