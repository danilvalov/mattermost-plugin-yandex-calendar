// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func TestUserEventSnapshotKey(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		require.Equal(t, "", UserEventSnapshotKey(nil))
	})

	t.Run("ical only when no start", func(t *testing.T) {
		ev := &remote.Event{ICalUID: "uid@x", ID: "graph-id"}
		require.Equal(t, "uid@x", UserEventSnapshotKey(ev))
	})

	t.Run("id fallback when ical empty", func(t *testing.T) {
		ev := &remote.Event{ID: "only-id"}
		require.Equal(t, "only-id", UserEventSnapshotKey(ev))
	})

	t.Run("ical plus start disambiguates recurrence", func(t *testing.T) {
		ical := "series@google.com"
		a := &remote.Event{
			ICalUID: ical,
			Start:    remote.NewDateTime(time.Date(2026, 5, 27, 17, 0, 0, 0, time.FixedZone("+6", 6*3600)), "Asia/Omsk"),
		}
		b := &remote.Event{
			ICalUID: ical,
			Start:    remote.NewDateTime(time.Date(2026, 7, 27, 17, 0, 0, 0, time.FixedZone("+6", 6*3600)), "Asia/Omsk"),
		}
		ka := UserEventSnapshotKey(a)
		kb := UserEventSnapshotKey(b)
		require.NotEqual(t, ka, kb)
		require.Contains(t, ka, ical+"|")
		require.Contains(t, kb, ical+"|")
	})
}

func TestTryReserveNotification_duplicateKeyReturnsFalseNil(t *testing.T) {
	mockAPI, st, _, _, _ := GetMockSetup(t)
	mockAPI.On("KVSetWithOptions", mock.Anything, []byte("1"), mock.Anything).Return(false, &model.AppError{
		Message: "duplicate key value violates unique constraint \"pluginkeyvaluestore_pkey\"",
	}).Times(1)

	ok, err := st.TryReserveNotification("user_key", time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
	mockAPI.AssertExpectations(t)
}
