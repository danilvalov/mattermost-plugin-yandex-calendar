// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	mattermostplugin "github.com/mattermost/mattermost/server/public/plugin"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/plugin"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/ycal"
)

var BuildHash string
var BuildHashShort string
var BuildDate string
var CalendarProvider string

func main() {
	config.Provider = ycal.GetYCalProviderConfig()

	mattermostplugin.ClientMain(
		plugin.NewWithEnv(
			engine.Env{
				Config: &config.Config{
					PluginID:       manifest.Id,
					PluginVersion:  manifest.Version,
					BuildHash:      BuildHash,
					BuildHashShort: BuildHashShort,
					BuildDate:      BuildDate,
					Provider:       config.Provider,
				},
				Dependencies: &engine.Dependencies{},
			}))
}
