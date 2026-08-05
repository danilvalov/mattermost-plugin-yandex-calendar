# Configure the Mattermost Yandex Calendar integration

In Mattermost, configure [Yandex Calendar integration](about.md) by going to **System Console > Plugins > Yandex Calendar**, entering the following values, and selecting **Save**.

## Settings

- **Enable Plugin**: Select `true` to enable the plugin for your Mattermost instance. Default `false`.
- **Admin User IDs**: Specify the user IDs who are authorized to manage the plugin in addition to Mattermost system admins. Separate multiple user IDs with commas. Go to **System Console > User Management > Users** to obtain a user’s ID.
- **Copy plugin logs to admins, as bot messages**: The level of detail in log events for the plugin. Can be one of: **None**, **Debug**, **Info**, **Warning**, or **Error**.
- **Display full context for each admin log message**: Specify whether full context is displayed for log messages.
- **Enable Calendar product UI**: When enabled, the Calendar product appears in the Mattermost product switcher. Disable to hide the UI while keeping slash commands and notifications. Users may need to reload the page after changing this.
- **Encryption key**: Generate an encryption key used to store credentials and data in the database. Regenerating this value clears stored authentication; users must connect again with `/ycal connect`.

## Usage statistics

Below the configuration fields, system admins see a read-only **Usage** panel. It does not change plugin config and does not enable **Save** by itself.

```
┌ Usage ──────────────────────────────────────────────────────────┐
│ Connected users are counted from stored Yandex Calendar creds. │
│                                                                 │
│ ┌────┐ ┌────┐ ┌────┐ ┌────┐   Connected · Adoption · Inactive · │
│ │ 12 │ │ 8% │ │  1 │ │  5 │   Subscriptions                     │
│ └────┘ └────┘ └────┘ └────┘                                     │
│ ┌────┐ ┌────┐ ┌────┐ ┌────┐   Reminders · Daily summary ·       │
│ │  9 │ │  4 │ │  7 │ │  6 │   Custom status · Auto status       │
│ └────┘ └────┘ └────┘ └────┘                                     │
│ ┌────┐ ┌────┐ ┌────┐ ┌────┐   Linked · In a meeting · Away · DND│
│ │  2 │ │  1 │ │  3 │ │  3 │                                     │
│ └────┘ └────┘ └────┘ └────┘                                     │
│ 12 of 150 Mattermost users have connected Yandex Calendar.      │
│ · Status not set: 6                                             │
└─────────────────────────────────────────────────────────────────┘
```

| Metric | Meaning |
|--------|---------|
| **Connected** | Users with a stored calendar token (`OAuth2Token` / CalDAV credentials present) |
| **Adoption** | Connected ÷ active human Mattermost users (bots and deleted users excluded). Shows `—` if Mattermost user stats cannot be loaded |
| **Inactive** | Stored user records without a token (need reconnect) |
| **Subscriptions** | Users with an event subscription ID set |
| **Reminders** / **Daily summary** / **Custom status** / **Auto status** | Feature adoption counts from per-user settings (auto status = Away + Do not disturb) |
| **Linked to channel** / **In a meeting now** | Users with channel↔event links / currently tracked active meetings |
| **Away** / **Do not disturb** | Status-update option buckets; “Status not set” is in the footer |

**Data sources**

- Plugin: `GET /plugins/mattermost-plugin-yandex-calendar/api/v1/admin/stats` — aggregates from the plugin KV user index (no PII in the response). Authorized for Mattermost system admins and users listed in **Admin User IDs**.
- Mattermost (adoption denominator only): `Client4.getFilteredUsersStats({ include_bots: false, include_deleted: false })`.

The same aggregate counts are also exposed as tiles on **System Console > Site Statistics** via the plugin site-stats handler.

## Troubleshooting

If your Mattermost users encounter issues when connecting calendars, creating events, inviting guests to events, or linking channels, we recommend restarting the plugin as a Mattermost system admin.

1. Go to **System Console > Plugins > Plugin Management**.

2. Under **Installed Plugins**, scroll to the **Yandex Calendar** section, select **Disable**, then wait for the **State** to change to **Not running**.

3. Select **Enable** and wait for the **State** to change to **Running**.

See [usage.md](usage.md) for how to use the integration after it is configured.
