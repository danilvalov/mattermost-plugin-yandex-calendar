// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package views

import (
	mmi18n "github.com/mattermost/mattermost/server/public/pluginapi/i18n"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/locale"
)

func tr(b *mmi18n.Bundle, userID, id, defaultOther string, data map[string]any) string {
	return locale.User(b, userID, id, defaultOther, data)
}
