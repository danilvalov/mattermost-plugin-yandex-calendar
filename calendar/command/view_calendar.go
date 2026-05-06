// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"strings"
	"time"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine/views"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
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

	now := time.Now()
	startOfCurrentDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	events, err := c.Engine.ViewCalendar(c.user(), startOfCurrentDay, now.Add(14*24*time.Hour))
	if err != nil {
		return "", false, err
	}
	events = filterOngoingAndUpcomingEvents(events, now)

	out, err := views.RenderCalendarViewWithTimeFormat(events, tz, c.Engine.IsMilitaryTime(c.user()), c.I18n, c.Args.UserId)
	return out, false, err
}

func filterOngoingAndUpcomingEvents(events []*remote.Event, now time.Time) []*remote.Event {
	filtered := make([]*remote.Event, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		if !event.End.Time().Before(now) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}
