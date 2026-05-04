// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"strings"
	"time"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
)

func (c *Command) dailySummaryHelp() string {
	return c.T("ycal.daily.help", "### Daily summary commands:\n"+
		"`/{{.Trigger}} summary view` - View your daily summary\n"+
		"`/{{.Trigger}} summary settings` - View your settings for the daily summary\n"+
		"`/{{.Trigger}} summary time 8:00AM` - Set the time you would like to receive your daily summary\n"+
		"`/{{.Trigger}} summary enable` - Enable your daily summary\n"+
		"`/{{.Trigger}} summary disable` - Disable your daily summary",
		map[string]any{"Trigger": config.Provider.CommandTrigger})
}

func (c *Command) dailySummarySetTimeHint() string {
	return c.T("ycal.daily.set_time_hint", "Please enter a time, for example:\n`/{{.Trigger}} summary time 8:00AM`",
		map[string]any{"Trigger": config.Provider.CommandTrigger})
}

func (c *Command) dailySummary(parameters ...string) (string, bool, error) {
	if len(parameters) == 0 {
		return c.dailySummaryHelp(), false, nil
	}

	switch parameters[0] {
	case "view", "today":
		postStr, err := c.Engine.GetDaySummaryForUser(time.Now(), c.user())
		if err != nil {
			if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
				return c.T("ycal.err.user_inactive", store.ErrorUserInactive, nil), false, nil
			}

			return err.Error(), false, err
		}
		return postStr, false, nil
	case "tomorrow":
		postStr, err := c.Engine.GetDaySummaryForUser(time.Now().Add(time.Hour*24), c.user())
		if err != nil {
			if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
				return c.T("ycal.err.user_inactive", store.ErrorUserInactive, nil), false, nil
			}

			return err.Error(), false, err
		}
		return postStr, false, nil
	case "time":
		if len(parameters) != 2 {
			return c.dailySummarySetTimeHint(), false, nil
		}
		val := parameters[1]

		dsum, err := c.Engine.SetDailySummaryPostTime(c.user(), val)
		if err != nil {
			if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
				return c.T("ycal.err.user_inactive", store.ErrorUserInactive, nil), false, nil
			}

			return err.Error() + "\n" + c.dailySummarySetTimeHint(), false, nil
		}

		return c.dailySummaryResponse(dsum), false, nil
	case "settings":
		dsum, err := c.Engine.GetDailySummarySettingsForUser(c.user())
		if err != nil {
			if strings.Contains(err.Error(), store.ErrorRefreshTokenNotSet) || strings.Contains(err.Error(), store.ErrorUserInactive) {
				return c.T("ycal.err.user_inactive", store.ErrorUserInactive, nil), false, nil
			}

			suffix := c.T("ycal.daily.configure_suffix", "You may need to configure your daily summary using the commands below.\n{{.Help}}",
				map[string]any{"Help": c.dailySummaryHelp()})
			return err.Error() + "\n" + suffix, false, nil
		}

		return c.dailySummaryResponse(dsum), false, nil
	case "enable":
		dsum, err := c.Engine.SetDailySummaryEnabled(c.user(), true)
		if err != nil {
			return err.Error(), false, err
		}

		return c.dailySummaryResponse(dsum), false, nil
	case "disable":
		dsum, err := c.Engine.SetDailySummaryEnabled(c.user(), false)
		if err != nil {
			return err.Error(), false, err
		}
		return c.dailySummaryResponse(dsum), false, nil
	}
	return c.T("ycal.daily.invalid", "Invalid command. Please try again\n\n{{.Help}}",
		map[string]any{"Help": c.dailySummaryHelp()}), false, nil
}

func (c *Command) dailySummaryResponse(dsum *store.DailySummaryUserSettings) string {
	if dsum.PostTime == "" {
		return c.T("ycal.daily.need_time", "Your daily summary time is not yet configured.\n{{.Hint}}",
			map[string]any{"Hint": c.dailySummarySetTimeHint()})
	}

	disableHint := ""
	if !dsum.Enable {
		disableHint = c.T("ycal.daily.disabled_suffix", ", but is disabled. Enable it with `/{{.Trigger}} summary enable`",
			map[string]any{"Trigger": config.Provider.CommandTrigger})
	}
	return c.T("ycal.daily.state", "Your daily summary is configured to show at {{.PostTime}} {{.Timezone}}{{.DisableHint}}.",
		map[string]any{
			"PostTime":    dsum.PostTime,
			"Timezone":    dsum.Timezone,
			"DisableHint": disableHint,
		})
}
