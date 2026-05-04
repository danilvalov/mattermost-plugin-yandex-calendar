// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine/mock_plugin_api"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine/mock_welcomer"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store/mock_store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot/mock_bot"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/ycal"
)

// This plugin uses CalDAV app passwords only; OAuth init must be rejected when
// PasswordAuth is enabled (no Microsoft/Google OAuth in tests).
func TestInitOAuth2_RejectsWhenPasswordAuth(t *testing.T) {
	saved := config.Provider
	t.Cleanup(func() { config.Provider = saved })
	config.Provider = ycal.GetYCalProviderConfig()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		PluginURL: "http://localhost",
		Provider:  config.Provider,
	}
	env := Env{
		Config: cfg,
		Dependencies: &Dependencies{
			Store:             mock_store.NewMockStore(ctrl),
			Logger:            &bot.NilLogger{},
			Poster:            mock_bot.NewMockPoster(ctrl),
			Remote:            remote.Makers[ycal.Kind](cfg, &bot.NilLogger{}),
			PluginAPI:         mock_plugin_api.NewMockPluginAPI(ctrl),
			Welcomer:          mock_welcomer.NewMockWelcomer(ctrl),
			IsAuthorizedAdmin: func(string) (bool, error) { return false, nil },
		},
	}

	app := NewOAuth2App(env)
	_, err := app.InitOAuth2("user@mattermost.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "CalDAV")
}
