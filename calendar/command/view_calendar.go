// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"strings"
	"time"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine/views"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
)

func (c *Command) viewCalendar(_ ...string) (string, bool, error) {
	tz, err := c.Engine.GetTimezone(c.user())
	if err != nil {
		if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
			return c.T("ycal.err.user_inactive", store.ErrorUserInactive, nil), false, nil
		}

		return c.T("ycal.viewcal.no_timezone", "Error: No timezone found", nil), false, err
	}

	startOfCurrentDay := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	events, err := c.Engine.ViewCalendar(c.user(), startOfCurrentDay, time.Now().Add(14*24*time.Hour))
	if err != nil {
		return "", false, err
	}

	out, err := views.RenderCalendarView(events, tz, c.I18n, c.Args.UserId)
	return out, false, err
}
