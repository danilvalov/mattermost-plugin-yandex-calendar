// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
)

const (
	ConnectBotAlreadyConnectedTemplate = "The bot account is already connected to %s account `%s`. To connect to a different account, first run `/%s disconnect_bot`."
	ConnectBotSuccessTemplate          = "[Click here to link the bot's %s account.](%s/oauth2/connect_bot)"
)

func (c *Command) connect(_ ...string) (string, bool, error) {
	ru, err := c.Engine.GetRemoteUser(c.Args.UserId)
	if err == nil {
		return c.T("ycal.connect.already_connected",
			"Your Mattermost account is already connected to {{.DisplayName}} account `{{.Mail}}`. To connect to a different account, first run `/{{.Trigger}} disconnect`.",
			map[string]any{
				"DisplayName": config.Provider.DisplayName,
				"Mail":        ru.Mail,
				"Trigger":     config.Provider.CommandTrigger,
			}), false, nil
	}

	out := ""

	err = c.Engine.Welcome(c.Args.UserId)
	if err != nil {
		out = c.T("ycal.connect.error", "There has been a problem while trying to connect: {{.Err}}",
			map[string]any{"Err": err.Error()})
	}

	return out, true, nil
}
