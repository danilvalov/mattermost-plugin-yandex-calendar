// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package msgraph

import (
	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

// UpdateEvent is unused by the Yandex CalDAV plugin path; stub satisfies remote.Client.
func (c *client) UpdateEvent(_ *remote.Event) (*remote.Event, error) {
	return nil, errors.New("msgraph UpdateEvent: not implemented")
}

// DeleteEvent is unused by the Yandex CalDAV plugin path; stub satisfies remote.Client.
func (c *client) DeleteEvent(_ string) error {
	return errors.New("msgraph DeleteEvent: not implemented")
}
