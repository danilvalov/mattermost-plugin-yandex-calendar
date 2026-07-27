// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"net/http"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/httputils"
)

func (api *api) connectedUserHandler(w http.ResponseWriter, r *http.Request) {
	user, err := api.loadConnectedUser(r)
	if err != nil {
		api.writeStoreOrAuthError(w, err)
		return
	}

	email := ""
	if user.Remote != nil {
		email = user.Remote.Mail
	}
	timezone := mattermostUserTimezone(api, user.MattermostUserID, "")

	_ = httputils.WriteJSONResponse(w, map[string]any{
		"is_connected": true,
		"email":        email,
		"timezone":     timezone,
	}, http.StatusOK)
}
