// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

// RunPollNotificationsJob polls each connected user and enqueues event notifications.
func RunPollNotificationsJob(env Env) {
	if env.NotificationProcessor == nil {
		return
	}
	if !config.Provider.Features.EnableEventPolling || !config.Provider.Features.EventNotifications {
		return
	}

	uindex, err := env.Store.LoadUserIndex()
	if err != nil {
		env.Logger.Errorf("Poll job failed to load user index: %v", err)
		return
	}

	for _, us := range uindex {
		msc, ok := New(env, us.MattermostUserID).(*mscalendar)
		if !ok {
			continue
		}
		if err := msc.Filter(withActingUserExpanded, withClient); err != nil {
			env.Logger.Debugf("Poll job skip user %s: %v", us.MattermostUserID, err)
			continue
		}
		user, err := env.Store.LoadUser(us.MattermostUserID)
		if err != nil || user.Settings.EventSubscriptionID == "" {
			continue
		}

		notifs, err := msc.client.PollNotifications(us.RemoteID, user.Settings.EventSubscriptionID)
		if err != nil {
			if remote.IsUnauthorized(err) {
				disconnectUnauthorizedPollUser(env, msc, us.MattermostUserID, us.RemoteID, err)
				continue
			}
			env.Logger.Errorf("PollNotifications for %s: %v", us.MattermostUserID, err)
			continue
		}

		for _, n := range notifs {
			if n == nil {
				continue
			}
			if err := env.NotificationProcessor.Enqueue(n); err != nil {
				env.Logger.Warnf("Poll job enqueue: %v", err)
			}
		}
	}
}

// disconnectUnauthorizedPollUser drops a user whose CalDAV credentials no longer work
// (e.g. app password revoked), same outcome as `/ycal disconnect`.
func disconnectUnauthorizedPollUser(env Env, msc *mscalendar, mattermostUserID, remoteID string, cause error) {
	env.Logger.Warnf("PollNotifications auth failed for %s, disconnecting: %v", mattermostUserID, cause)

	subID := ""
	if u, err := env.Store.LoadUser(mattermostUserID); err == nil {
		subID = u.Settings.EventSubscriptionID
	}

	if err := msc.DisconnectUser(mattermostUserID); err != nil {
		env.Logger.Errorf("Disconnect after auth failure for %s: %v; force-deleting", mattermostUserID, err)
		if ferr := env.Store.ForceDeleteUser(mattermostUserID, remoteID); ferr != nil {
			env.Logger.Errorf("ForceDeleteUser after auth failure for %s: %v", mattermostUserID, ferr)
			return
		}
		// ForceDelete skips subscription KV — clean orphan if we still know the id.
		if subID != "" {
			if derr := env.Store.DeleteUserSubscription(nil, subID); derr != nil {
				env.Logger.Warnf("orphan subscription cleanup %s after auth failure: %v", subID, derr)
			}
		}
	}

	msc.ClearSettingsPosts(mattermostUserID)

	msg := env.Tr(mattermostUserID, "ycal.disconnect.auth_failed",
		"Your {{.DisplayName}} account was disconnected because authentication failed (the app password may have been revoked). Run `/{{.Trigger}} connect` to reconnect.",
		map[string]any{
			"DisplayName": config.Provider.DisplayName,
			"Trigger":     config.Provider.CommandTrigger,
		})
	// Poster.DM treats the message as fmt format — pass via %s.
	if _, err := env.Poster.DM(mattermostUserID, "%s", msg); err != nil {
		env.Logger.Warnf("DM after auth disconnect for %s: %v", mattermostUserID, err)
	}
}
