// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"testing"
	"time"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/stretchr/testify/require"
)

func TestFilterOngoingAndUpcomingEvents(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)

	past := &remote.Event{
		Subject: "Past",
		Start:   remote.NewDateTime(now.Add(-2*time.Hour), "UTC"),
		End:     remote.NewDateTime(now.Add(-1*time.Hour), "UTC"),
	}
	ongoing := &remote.Event{
		Subject: "Ongoing",
		Start:   remote.NewDateTime(now.Add(-30*time.Minute), "UTC"),
		End:     remote.NewDateTime(now.Add(30*time.Minute), "UTC"),
	}
	upcoming := &remote.Event{
		Subject: "Upcoming",
		Start:   remote.NewDateTime(now.Add(2*time.Hour), "UTC"),
		End:     remote.NewDateTime(now.Add(3*time.Hour), "UTC"),
	}

	got := filterOngoingAndUpcomingEvents([]*remote.Event{past, ongoing, upcoming, nil}, now)

	require.Equal(t, []*remote.Event{ongoing, upcoming}, got)
}
