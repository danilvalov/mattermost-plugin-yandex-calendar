package ycal

import (
	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func (c *client) CreateCalendar(calIn *remote.Calendar) (*remote.Calendar, error) {
	return nil, errors.New("ycal CreateCalendar not supported")
}

func (c *client) DeleteCalendar(calID string) error {
	return errors.New("ycal DeleteCalendar not supported")
}

func (c *client) GetCalendars(remoteUserID string) ([]*remote.Calendar, error) {
	return nil, errors.New("ycal GetCalendars not implemented")
}

func (c *client) GetDefaultCalendar() (*remote.Calendar, error) {
	return &remote.Calendar{
		ID:   "default",
		Name: "Default",
	}, nil
}
