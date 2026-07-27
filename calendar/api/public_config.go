package api

import (
	"net/http"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/httputils"
)

func (api *api) publicConfig(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Mattermost-User-Id") == "" {
		httputils.WriteUnauthorizedError(w, errUnauthorized)
		return
	}
	_ = httputils.WriteJSONResponse(w, map[string]any{
		"enable_calendar_ui": api.Config.CalendarUIEnabled(),
	}, http.StatusOK)
}
