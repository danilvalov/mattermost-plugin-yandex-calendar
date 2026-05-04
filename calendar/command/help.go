// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"fmt"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
)

func (c *Command) help(_ ...string) (string, bool, error) {
	cmds := getCommands(func(id, def string, data map[string]any) string {
		return c.T(id, def, data)
	})
	resp := ""
	for _, cmd := range cmds {
		desc := cmd.Trigger
		if cmd.HelpText != "" {
			desc += " - " + cmd.HelpText
		}
		resp += getCommandText(desc)
	}

	return resp, false, nil
}

func getCommandText(s string) string {
	return fmt.Sprintf("/%s %s\n", config.Provider.CommandTrigger, s)
}
