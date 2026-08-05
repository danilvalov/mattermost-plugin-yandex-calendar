// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package store

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/testutil"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot/mock_bot"
)

func TestGetStats(t *testing.T) {
	const (
		user1ID  = "user1"
		user2ID  = "user2"
		user1Key = "user_24c9e15e52afc47c225b757e7bee1f9d"
		user2Key = "user_7e58d63b60197ceb55a1c487989a3720"
	)

	mustJSON := func(v interface{}) []byte {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return b
	}

	tests := []struct {
		name       string
		setup      func(*testutil.MockPluginAPI, *mock_bot.MockLogger, *mock_bot.MockLogger)
		assertions func(*testing.T, *Stats, error)
	}{
		{
			name: "empty index returns zeros",
			setup: func(mockAPI *testutil.MockPluginAPI, _, _ *mock_bot.MockLogger) {
				mockAPI.On("KVGet", "userindex_").Return([]byte(`[]`), nil)
			},
			assertions: func(t *testing.T, stats *Stats, err error) {
				require.NoError(t, err)
				require.Equal(t, &Stats{}, stats)
			},
		},
		{
			name: "missing index key returns zeros",
			setup: func(mockAPI *testutil.MockPluginAPI, _, _ *mock_bot.MockLogger) {
				mockAPI.On("KVGet", "userindex_").Return(nil, nil)
			},
			assertions: func(t *testing.T, stats *Stats, err error) {
				require.NoError(t, err)
				require.Equal(t, &Stats{}, stats)
			},
		},
		{
			name: "index load error",
			setup: func(mockAPI *testutil.MockPluginAPI, _, _ *mock_bot.MockLogger) {
				mockAPI.On("KVGet", "userindex_").Return(nil, &model.AppError{Message: "Load failed"})
			},
			assertions: func(t *testing.T, stats *Stats, err error) {
				require.Nil(t, stats)
				require.Error(t, err)
			},
		},
		{
			name: "mixed users with feature flags and skip load error",
			setup: func(mockAPI *testutil.MockPluginAPI, mockLogger, mockLoggerWith *mock_bot.MockLogger) {
				index := UserIndex{
					{MattermostUserID: user1ID},
					{MattermostUserID: user2ID},
					{MattermostUserID: "missing"},
				}
				mockAPI.On("KVGet", "userindex_").Return(mustJSON(index), nil)

				u1 := &User{
					MattermostUserID: user1ID,
					OAuth2Token:      &oauth2.Token{AccessToken: "tok"},
					Settings: Settings{
						ReceiveReminders:        true,
						SetCustomStatus:         true,
						EventSubscriptionID:     "sub-1",
						UpdateStatusFromOptions: AwayStatusOption,
						DailySummary:            &DailySummaryUserSettings{Enable: true},
					},
					ChannelEvents: ChannelEventLink{"ev1": "ch1"},
					ActiveEvents:  []string{"ev1"},
				}
				u2 := &User{
					MattermostUserID: user2ID,
					OAuth2Token:      nil,
					Settings: Settings{
						UpdateStatus:                      true,
						ReceiveNotificationsDuringMeeting: false, // legacy → DND
					},
				}
				mockAPI.On("KVGet", user1Key).Return(mustJSON(u1), nil)
				mockAPI.On("KVGet", user2Key).Return(mustJSON(u2), nil)
				mockAPI.On("KVGet", "user_ea21841da70e6405af19fabc4ff8bdd9").Return(nil, &model.AppError{Message: "missing"})

				mockLogger.EXPECT().With(map[string]interface{}{
					"mm_user_id": "missing",
					"error":      "failed plugin KVGet: missing",
				}).Return(mockLoggerWith).Times(1)
				mockLoggerWith.EXPECT().Debugf("store: GetStats skipping user").Times(1)
			},
			assertions: func(t *testing.T, stats *Stats, err error) {
				require.NoError(t, err)
				require.Equal(t, uint64(1), stats.ConnectedUsers)
				require.Equal(t, uint64(1), stats.InactiveUsers)
				require.Equal(t, uint64(1), stats.Subscriptions)
				require.Equal(t, uint64(1), stats.ReceiveReminders)
				require.Equal(t, uint64(1), stats.DailySummaryEnabled)
				require.Equal(t, uint64(1), stats.SetCustomStatus)
				require.Equal(t, uint64(1), stats.StatusAway)
				require.Equal(t, uint64(1), stats.StatusDND)
				require.Equal(t, uint64(0), stats.StatusNotSet)
				require.Equal(t, uint64(1), stats.WithChannelEvents)
				require.Equal(t, uint64(1), stats.WithActiveEvents)

				n := stats.ConnectedUsers + stats.InactiveUsers
				require.Equal(t, n, stats.StatusAway+stats.StatusDND+stats.StatusNotSet)
			},
		},
		{
			name: "legacy UpdateStatus with notifications during meeting maps to Away",
			setup: func(mockAPI *testutil.MockPluginAPI, _, _ *mock_bot.MockLogger) {
				index := UserIndex{{MattermostUserID: user1ID}}
				mockAPI.On("KVGet", "userindex_").Return(mustJSON(index), nil)
				u := &User{
					MattermostUserID: user1ID,
					OAuth2Token:      &oauth2.Token{AccessToken: "tok"},
					Settings: Settings{
						UpdateStatus:                      true,
						ReceiveNotificationsDuringMeeting: true,
					},
				}
				mockAPI.On("KVGet", user1Key).Return(mustJSON(u), nil)
			},
			assertions: func(t *testing.T, stats *Stats, err error) {
				require.NoError(t, err)
				require.Equal(t, uint64(1), stats.ConnectedUsers)
				require.Equal(t, uint64(1), stats.StatusAway)
				require.Equal(t, uint64(0), stats.StatusDND)
				require.Equal(t, uint64(0), stats.StatusNotSet)
			},
		},
		{
			name: "explicit NotSet status",
			setup: func(mockAPI *testutil.MockPluginAPI, _, _ *mock_bot.MockLogger) {
				index := UserIndex{{MattermostUserID: user1ID}}
				mockAPI.On("KVGet", "userindex_").Return(mustJSON(index), nil)
				u := &User{
					MattermostUserID: user1ID,
					OAuth2Token:      &oauth2.Token{AccessToken: "tok"},
					Settings: Settings{
						UpdateStatusFromOptions: NotSetStatusOption,
					},
				}
				mockAPI.On("KVGet", user1Key).Return(mustJSON(u), nil)
			},
			assertions: func(t *testing.T, stats *Stats, err error) {
				require.NoError(t, err)
				require.Equal(t, uint64(1), stats.StatusNotSet)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI, store, mockLogger, mockLoggerWith, _ := GetMockSetup(t)
			tt.setup(mockAPI, mockLogger, mockLoggerWith)

			stats, err := store.GetStats()
			tt.assertions(t, stats, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestStatusUpdateBucket(t *testing.T) {
	tests := []struct {
		name string
		user *User
		want string
	}{
		{
			name: "away option",
			user: &User{Settings: Settings{UpdateStatusFromOptions: AwayStatusOption}},
			want: AwayStatusOption,
		},
		{
			name: "dnd option",
			user: &User{Settings: Settings{UpdateStatusFromOptions: DNDStatusOption}},
			want: DNDStatusOption,
		},
		{
			name: "legacy update status with notifications → away",
			user: &User{Settings: Settings{UpdateStatus: true, ReceiveNotificationsDuringMeeting: true}},
			want: AwayStatusOption,
		},
		{
			name: "legacy update status without notifications → dnd",
			user: &User{Settings: Settings{UpdateStatus: true}},
			want: DNDStatusOption,
		},
		{
			name: "not set",
			user: &User{Settings: Settings{UpdateStatusFromOptions: NotSetStatusOption}},
			want: NotSetStatusOption,
		},
		{
			name: "empty defaults to not set",
			user: &User{},
			want: NotSetStatusOption,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, statusUpdateBucket(tt.user))
			// Ensure read-only: legacy path must not mutate.
			if tt.user.Settings.UpdateStatusFromOptions == "" && tt.user.Settings.UpdateStatus {
				require.Empty(t, tt.user.Settings.UpdateStatusFromOptions)
			}
		})
	}
}
