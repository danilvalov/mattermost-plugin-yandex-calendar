package api

import (
	"net/http"
	"strings"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/httputils"
	"github.com/mattermost/mattermost/server/public/model"
)

type mmUserSearchHit struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

func (api *api) searchMMUsers(w http.ResponseWriter, r *http.Request) {
	user, err := api.loadConnectedUser(r)
	if err != nil {
		api.writeStoreOrAuthError(w, err)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		_ = httputils.WriteJSONResponse(w, []mmUserSearchHit{}, http.StatusOK)
		return
	}

	teams, err := api.PluginAPI.GetMattermostUserTeams(user.MattermostUserID)
	if err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("searchMMUsers teams")
		httputils.WriteInternalServerError(w, err)
		return
	}
	if len(teams) == 0 {
		_ = httputils.WriteJSONResponse(w, []mmUserSearchHit{}, http.StatusOK)
		return
	}

	const limit = 20
	seen := map[string]struct{}{}
	users := make([]*model.User, 0, limit)
	for _, team := range teams {
		if team == nil || team.Id == "" {
			continue
		}
		hits, searchErr := api.PluginAPI.SearchUsers(q, limit, team.Id)
		if searchErr != nil {
			api.Logger.With(bot.LogContext{"err": searchErr.Error(), "team_id": team.Id}).Errorf("searchMMUsers")
			httputils.WriteInternalServerError(w, searchErr)
			return
		}
		for _, u := range hits {
			if u == nil || u.Id == "" {
				continue
			}
			if _, ok := seen[u.Id]; ok {
				continue
			}
			seen[u.Id] = struct{}{}
			users = append(users, u)
			if len(users) >= limit {
				break
			}
		}
		if len(users) >= limit {
			break
		}
	}

	out := make([]mmUserSearchHit, 0, len(users))
	for _, u := range users {
		if strings.TrimSpace(u.Email) == "" || u.DeleteAt != 0 {
			continue
		}
		name := strings.TrimSpace(u.GetFullName())
		if name == "" {
			name = u.Username
		}
		out = append(out, mmUserSearchHit{
			ID:          u.Id,
			Username:    u.Username,
			DisplayName: name,
			Email:       u.Email,
		})
	}
	_ = httputils.WriteJSONResponse(w, out, http.StatusOK)
}
