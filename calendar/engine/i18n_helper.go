// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/locale"
)

// Tr returns a localized string for the given Mattermost user (English if bundle is nil).
func (e Env) Tr(userID, id, defaultOther string, data map[string]any) string {
	if e.Dependencies == nil {
		return locale.User(nil, userID, id, defaultOther, data)
	}
	return locale.User(e.I18n, userID, id, defaultOther, data)
}
