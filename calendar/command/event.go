// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package command

func (c *Command) event(parameters ...string) (string, bool, error) {
	if len(parameters) == 0 {
		return c.dailySummaryHelp(), false, nil
	}

	if parameters[0] == "create" {
		return c.T("ycal.event.desktop_only", "Creating events is only supported on desktop.", nil), false, nil
	}

	return "", false, nil
}
