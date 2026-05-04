// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"net/url"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
)

func (c *Command) info(_ ...string) (string, bool, error) {
	return c.T("ycal.info.version",
		"Mattermost {{.DisplayName}} plugin version: {{.Version}}, [{{.HashShort}}](https://github.com/mattermost/{{.Repo}}/commit/{{.Hash}}), built {{.Date}}\n",
		map[string]any{
			"DisplayName": c.Config.Provider.DisplayName,
			"Version":     c.Config.PluginVersion,
			"HashShort":   c.Config.BuildHashShort,
			"Repo":        url.PathEscape(config.Provider.Repository),
			"Hash":        url.PathEscape(c.Config.BuildHash),
			"Date":        c.Config.BuildDate,
		}), false, nil
}
