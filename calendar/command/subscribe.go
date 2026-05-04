// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

import (
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils"
)

func (c *Command) subscribe(parameters ...string) (string, bool, error) {
	if len(parameters) > 0 && parameters[0] == "list" {
		return c.debugList()
	}

	_, err := c.Engine.LoadMyEventSubscription()
	if err == nil {
		return c.T("ycal.subscribe.already", "You are already subscribed to events.", nil), false, nil
	}

	_, err = c.Engine.CreateMyEventSubscription()
	if err != nil {
		return "", false, err
	}
	return c.T("ycal.subscribe.success", "You are now subscribed to events.", nil), false, nil
}

func (c *Command) debugList() (string, bool, error) {
	subs, err := c.Engine.ListRemoteSubscriptions()
	if err != nil {
		return "", false, err
	}
	return c.T("ycal.subscribe.list_prefix", "Subscriptions:", nil) + utils.JSONBlock(subs), false, nil
}
