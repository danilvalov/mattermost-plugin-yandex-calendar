package ycal

import (
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func (c *client) GetSchedule(_ []*remote.ScheduleUserInfo, _, _ *remote.DateTime, _ int) ([]*remote.ScheduleInformation, error) {
	return nil, remote.ErrNotImplemented
}
