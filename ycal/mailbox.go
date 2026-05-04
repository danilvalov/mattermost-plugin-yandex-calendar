package ycal

import (
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func (c *client) GetMailboxSettings(remoteUserID string) (*remote.MailboxSettings, error) {
	// Yandex CalDAV does not expose mailbox timezone via a simple API here; use Moscow as common default.
	return &remote.MailboxSettings{
		TimeZone: "Europe/Moscow",
	}, nil
}
