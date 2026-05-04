// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

func (c *Command) unsubscribe(_ ...string) (string, bool, error) {
	_, err := c.Engine.LoadMyEventSubscription()
	if err != nil {
		return c.T("ycal.unsubscribe.not_subscribed", "You are not subscribed to events.", nil), false, nil
	}

	err = c.Engine.DeleteMyEventSubscription()
	if err != nil {
		return "", false, err
	}

	return c.T("ycal.unsubscribe.success", "You have unsubscribed from events.", nil), false, nil
}
