# Use the Mattermost Yandex Calendar integration

The plugin offers slash commands, settings, daily summary, status updates, and reminders familiar from Mattermost calendar plugins, with data loaded from Yandex Calendar via CalDAV.

## Connect

1. Run `/ycal connect` and open the **connect page** from the bot message.
2. Enter your Yandex account email and **app password** (see [setup.md](setup.md)).

## Commands

The command prefix is `/ycal` (for example `/ycal today`, `/ycal settings`). Use `/ycal help` in Mattermost for the full list supported by the calendar engine.

## Create event (channel menu)

When enabled in the webapp, use **Create calendar event** from the channel menu. Creating events through the API may require follow-up work depending on your Yandex account permissions; the server currently focuses on read/sync behaviour.
