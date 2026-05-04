// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
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
