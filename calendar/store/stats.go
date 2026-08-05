// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package store

import "errors"

// Stats is an aggregate snapshot of connected users and feature adoption from KV.
type Stats struct {
	ConnectedUsers      uint64 `json:"connected_users"`       // OAuth2Token != nil
	InactiveUsers       uint64 `json:"inactive_users"`        // loaded, token == nil
	Subscriptions       uint64 `json:"subscriptions"`         // EventSubscriptionID != ""

	ReceiveReminders    uint64 `json:"receive_reminders"`
	DailySummaryEnabled uint64 `json:"daily_summary_enabled"`
	SetCustomStatus     uint64 `json:"set_custom_status"`
	StatusAway          uint64 `json:"status_away"`
	StatusDND           uint64 `json:"status_dnd"`
	StatusNotSet        uint64 `json:"status_not_set"`

	WithChannelEvents uint64 `json:"with_channel_events"`
	WithActiveEvents  uint64 `json:"with_active_events"`
}

// GetStats aggregates usage metrics from the user index and per-user records.
// Load failures for individual users are skipped; a missing/empty index yields zeros.
func (s *pluginStore) GetStats() (*Stats, error) {
	stats := &Stats{}

	index, err := s.LoadUserIndex()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return stats, nil
		}
		return nil, err
	}
	if len(index) == 0 {
		return stats, nil
	}

	for _, short := range index {
		if short == nil || short.MattermostUserID == "" {
			continue
		}

		user, loadErr := s.LoadUser(short.MattermostUserID)
		if loadErr != nil {
			s.Logger.With(map[string]interface{}{
				"mm_user_id": short.MattermostUserID,
				"error":      loadErr.Error(),
			}).Debugf("store: GetStats skipping user")
			continue
		}

		accumulateUserStats(stats, user)
	}

	return stats, nil
}

func accumulateUserStats(stats *Stats, user *User) {
	if user.OAuth2Token != nil {
		stats.ConnectedUsers++
	} else {
		stats.InactiveUsers++
	}

	if user.Settings.EventSubscriptionID != "" {
		stats.Subscriptions++
	}
	if user.Settings.ReceiveReminders {
		stats.ReceiveReminders++
	}
	if user.Settings.DailySummary != nil && user.Settings.DailySummary.Enable {
		stats.DailySummaryEnabled++
	}
	if user.IsConfiguredForCustomStatusUpdates() {
		stats.SetCustomStatus++
	}

	switch statusUpdateBucket(user) {
	case AwayStatusOption:
		stats.StatusAway++
	case DNDStatusOption:
		stats.StatusDND++
	default:
		stats.StatusNotSet++
	}

	if len(user.ChannelEvents) > 0 {
		stats.WithChannelEvents++
	}
	if len(user.ActiveEvents) > 0 {
		stats.WithActiveEvents++
	}
}

// statusUpdateBucket returns Away, DND, or NotSet without mutating user settings.
// Mirrors IsConfiguredForStatusUpdates legacy mapping, but read-only.
func statusUpdateBucket(user *User) string {
	opt := user.Settings.UpdateStatusFromOptions
	if opt == AwayStatusOption || opt == DNDStatusOption {
		return opt
	}

	if opt == "" && user.Settings.UpdateStatus {
		if user.Settings.ReceiveNotificationsDuringMeeting {
			return AwayStatusOption
		}
		return DNDStatusOption
	}

	return NotSetStatusOption
}
