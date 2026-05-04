package ycal

import (
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func (c *client) GetMe() (*remote.User, error) {
	return &remote.User{
		ID:                c.email,
		Mail:              c.email,
		DisplayName:       splitMail(c.email),
		UserPrincipalName: c.email,
	}, nil
}

func splitMail(email string) string {
	for i := range email {
		if email[i] == '@' {
			return email[:i]
		}
	}
	return email
}

func (c *client) GetSuperuserToken() (string, error) {
	return "", remote.ErrSuperUserClientNotSupported
}
