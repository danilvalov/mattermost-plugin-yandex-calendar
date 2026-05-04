# Set up the Mattermost Yandex Calendar integration

This plugin connects to [Yandex Calendar](https://calendar.yandex.com/) using **CalDAV** and an **app password** from Yandex ID (no OAuth).

## Admin steps

1. Install the plugin on your Mattermost server (**System Console > Plugins > Plugin Management**).
2. Enable the plugin and set an **encryption key** in the plugin settings (required for storing credentials).
3. Ensure the Mattermost server **Site URL** is configured; restart the plugin after changes.

## User steps: app password

1. Sign in to [Yandex ID](https://id.yandex.com/) security settings.
2. Create an **app password** for mail/calendar sync (the exact UI label may vary by account type).
3. In Mattermost, run `/ycal connect` and open the **connect page** link from the bot message.
4. Enter your **Yandex email** and the **app password** (not your normal login password).

CalDAV endpoint used by the plugin: `https://caldav.yandex.ru`.

For usage, see [usage.md](usage.md).
