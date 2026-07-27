// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine/mock_plugin_api"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine/mock_welcomer"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote/mock_remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store/mock_store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot/mock_bot"
)

func TestDisconnectUnauthorizedPollUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prev := config.Provider
	config.Provider = config.ProviderConfig{DisplayName: "Yandex Calendar", CommandTrigger: "ycal"}
	t.Cleanup(func() { config.Provider = prev })

	mockStore := mock_store.NewMockStore(ctrl)
	mockPoster := mock_bot.NewMockPoster(ctrl)
	mockRemote := mock_remote.NewMockRemote(ctrl)
	mockClient := mock_remote.NewMockClient(ctrl)
	mockLogger := mock_bot.NewMockLogger(ctrl)
	mockWelcomer := mock_welcomer.NewMockWelcomer(ctrl)
	mockPluginAPI := mock_plugin_api.NewMockPluginAPI(ctrl)

	env := Env{
		Config: &config.Config{},
		Dependencies: &Dependencies{
			Store:     mockStore,
			Poster:    mockPoster,
			Remote:    mockRemote,
			Logger:    mockLogger,
			Welcomer:  mockWelcomer,
			PluginAPI: mockPluginAPI,
		},
	}
	msc := &mscalendar{Env: env, client: mockClient}

	uid := "mm-user-1"
	remoteID := "remote-1"
	stored := &store.User{
		MattermostUserID: uid,
		Settings:         store.Settings{EventSubscriptionID: "sub-1"},
		ChannelEvents:    store.ChannelEventLink{},
	}
	sub := &store.Subscription{Remote: &remote.Subscription{}}

	mockLogger.EXPECT().Warnf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	// preload sub id, then DisconnectUser LoadUser
	mockStore.EXPECT().LoadUser(uid).Return(stored, nil).Times(2)
	mockWelcomer.EXPECT().AfterDisconnect(uid).Return(nil)
	mockStore.EXPECT().LoadSubscription("sub-1").Return(sub, nil)
	mockStore.EXPECT().DeleteUserSubscription(stored, "sub-1").Return(nil)
	mockClient.EXPECT().DeleteSubscription(sub.Remote).Return(nil)
	mockStore.EXPECT().DeleteUser(uid).Return(nil)
	mockStore.EXPECT().DeleteUserFromIndex(uid).Return(nil)
	mockPoster.EXPECT().DM(uid, "%s", gomock.Any()).DoAndReturn(func(_ string, _ string, args ...interface{}) (string, error) {
		require.Len(t, args, 1)
		msg, ok := args[0].(string)
		require.True(t, ok)
		require.Contains(t, msg, "/ycal connect")
		require.Contains(t, msg, "Yandex Calendar")
		return "", nil
	})

	disconnectUnauthorizedPollUser(env, msc, uid, remoteID, fmt.Errorf("401 Unauthorized"))
}

func TestDisconnectUnauthorizedPollUserForceDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prev := config.Provider
	config.Provider = config.ProviderConfig{DisplayName: "Yandex Calendar", CommandTrigger: "ycal"}
	t.Cleanup(func() { config.Provider = prev })

	mockStore := mock_store.NewMockStore(ctrl)
	mockPoster := mock_bot.NewMockPoster(ctrl)
	mockRemote := mock_remote.NewMockRemote(ctrl)
	mockClient := mock_remote.NewMockClient(ctrl)
	mockLogger := mock_bot.NewMockLogger(ctrl)
	mockWelcomer := mock_welcomer.NewMockWelcomer(ctrl)
	mockPluginAPI := mock_plugin_api.NewMockPluginAPI(ctrl)

	env := Env{
		Config: &config.Config{},
		Dependencies: &Dependencies{
			Store:     mockStore,
			Poster:    mockPoster,
			Remote:    mockRemote,
			Logger:    mockLogger,
			Welcomer:  mockWelcomer,
			PluginAPI: mockPluginAPI,
		},
	}
	msc := &mscalendar{Env: env, client: mockClient}

	uid := "mm-user-2"
	remoteID := "remote-2"
	stored := &store.User{
		MattermostUserID: uid,
		Settings:         store.Settings{EventSubscriptionID: "sub-orphan"},
	}

	mockLogger.EXPECT().Warnf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockStore.EXPECT().LoadUser(uid).Return(stored, nil) // preload sub id
	mockWelcomer.EXPECT().AfterDisconnect(uid).Return(nil)
	mockStore.EXPECT().LoadUser(uid).Return(nil, errors.New("decrypt failed")) // DisconnectUser
	mockStore.EXPECT().LoadUserFromIndex(uid).Return(nil, errors.New("missing"))
	mockStore.EXPECT().ForceDeleteUser(uid, remoteID).Return(nil)
	mockStore.EXPECT().DeleteUserSubscription(nil, "sub-orphan").Return(nil)
	mockPoster.EXPECT().DM(uid, "%s", gomock.Any()).Return("", nil)

	disconnectUnauthorizedPollUser(env, msc, uid, remoteID, fmt.Errorf("401 Unauthorized"))
}
