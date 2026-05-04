# Configure the Mattermost Yandex Calendar integration

In Mattermost, configure [Yandex Calendar integration](about.md) by going to **System Console > Plugin Management > Yandex Calendar**, entering the following values, and selecting **Save**.

- **Enable Plugin**: Select `true` to enable the plugin for your Mattermost instance. Default `false`.
- **Admin User IDs**: Specify the user IDs who are authorized to manage the plugin in addition to Mattermost system admins. Separate multiple user IDs with commas. Go to **System Console > User Management > Users** to obtain a user’s ID.
- **Copy plugin logs to admins, as bot messages**: The level of detail in log events for the plugin. Can be one of: **None**, **Debug**, **Info**, **Warning**, or **Error**.
- **Display full context for each admin log message**: Specify whether full context is displayed for log messages.
- **Encryption key**: Generate an encryption key used to store credentials and data in the database. Regenerating this value clears stored authentication; users must connect again with `/ycal connect`.

## Troubleshooting

If your Mattermost users encounter issues when connecting calendars, creating events, inviting guests to events, or linking channels, we recommend restarting the plugin as a Mattermost system admin.

1. Go to **System Console > Plugins > Plugin Management**.

2. Under **Installed Plugins**, scroll to the **Yandex Calendar** section, select **Disable**, then wait for the **State** to change to **Not running**.

3. Select **Enable** and wait for the **State** to change to **Running**.

See [usage.md](usage.md) for how to use the integration after it is configured.
