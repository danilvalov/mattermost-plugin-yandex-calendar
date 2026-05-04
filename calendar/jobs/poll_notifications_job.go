// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package jobs

import (
	"time"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine"
)

const pollJobID = "poll_notifications"

func NewPollNotificationsJob() RegisteredJob {
	return RegisteredJob{
		id:       pollJobID,
		interval: time.Minute,
		work: func(env engine.Env) {
			engine.RunPollNotificationsJob(env)
		},
	}
}
