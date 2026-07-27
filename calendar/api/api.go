// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"net/http"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/httputils"
)

type api struct {
	engine.Env
	engine.NotificationProcessor
}

// Init initializes the router.
func Init(h *httputils.Handler, env engine.Env, notificationProcessor engine.NotificationProcessor) {
	api := &api{
		Env:                   env,
		NotificationProcessor: notificationProcessor,
	}

	apiRouter := h.Router.PathPrefix(config.PathAPI).Subrouter()
	apiRouter.HandleFunc("/authorized", api.getAuthorized).Methods(http.MethodGet)

	notificationRouter := h.Router.PathPrefix(config.PathNotification).Subrouter()
	notificationRouter.HandleFunc(config.PathEvent, api.notification).Methods(http.MethodPost)

	postActionRouter := h.Router.PathPrefix(config.PathPostAction).Subrouter()
	postActionRouter.HandleFunc(config.PathAccept, api.postActionAccept).Methods(http.MethodPost)
	postActionRouter.HandleFunc(config.PathDecline, api.postActionDecline).Methods(http.MethodPost)
	postActionRouter.HandleFunc(config.PathTentative, api.postActionTentative).Methods(http.MethodPost)
	postActionRouter.HandleFunc(config.PathRespond, api.postActionRespond).Methods(http.MethodPost)
	postActionRouter.HandleFunc(config.PathConfirmStatusChange, api.postActionConfirmStatusChange).Methods(http.MethodPost)

	dialogRouter := h.Router.PathPrefix(config.PathAutocomplete).Subrouter()
	dialogRouter.HandleFunc(config.PathUsers, api.autocompleteConnectedUsers)

	apiRoutes := h.Router.PathPrefix(config.InternalAPIPath).Subrouter()
	eventsRouter := apiRoutes.PathPrefix(config.PathEvents).Subrouter()
	eventsRouter.HandleFunc("", api.listEvents).Methods(http.MethodGet)
	eventsRouter.HandleFunc("", api.patchEvent).Methods(http.MethodPatch)
	eventsRouter.HandleFunc("", api.deleteEvent).Methods(http.MethodDelete)
	eventsRouter.HandleFunc(config.PathCreate, api.createEvent).Methods(http.MethodPost)
	eventsRouter.HandleFunc(config.PathGet, api.getEvent).Methods(http.MethodGet)
	eventsRouter.HandleFunc(config.PathRespond, api.respondEvent).Methods(http.MethodPost)
	apiRoutes.HandleFunc(config.PathConnectedUser, api.connectedUserHandler)
	apiRoutes.HandleFunc(config.PathUserSearch, api.searchMMUsers).Methods(http.MethodGet)
	apiRoutes.HandleFunc(config.PathConfig, api.publicConfig).Methods(http.MethodGet)

	// Returns provider information for the plugin to use
	apiRoutes.HandleFunc(config.PathProvider, func(w http.ResponseWriter, r *http.Request) {
		httputils.WriteJSONResponse(w, config.Provider, http.StatusOK)
	})
}
