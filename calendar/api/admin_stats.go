// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"fmt"
	"net/http"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/httputils"
)

func (api *api) adminStats(w http.ResponseWriter, r *http.Request) {
	mmUserID := r.Header.Get("Mattermost-User-Id")
	if mmUserID == "" {
		httputils.WriteUnauthorizedError(w, fmt.Errorf("unauthorized"))
		return
	}

	ok, err := engine.New(api.Env, mmUserID).IsAuthorizedAdmin(mmUserID)
	if err != nil {
		httputils.WriteInternalServerError(w, err)
		return
	}
	if !ok {
		httputils.WriteForbiddenError(w, fmt.Errorf("forbidden"))
		return
	}

	stats, err := api.Store.GetStats()
	if err != nil {
		httputils.WriteInternalServerError(w, err)
		return
	}

	_ = httputils.WriteJSONResponse(w, stats, http.StatusOK)
}
