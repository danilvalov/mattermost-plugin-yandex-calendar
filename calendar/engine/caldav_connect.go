// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
)

const caldavTokenExpiry = 2099

// CompleteCalDAVConnect wires HTTP handlers to CalDAV app-password authentication.
func CompleteCalDAVConnect(env Env, authedUserID, email, appPassword string) error {
	app := &oauth2App{Env: env}
	return app.completeCalDAVConnect(authedUserID, email, appPassword)
}

func (app *oauth2App) completeCalDAVConnect(authedUserID, email, appPassword string) error {
	if authedUserID == "" || email == "" || appPassword == "" {
		return errors.New("missing user, email or app password")
	}

	if _, err := app.Store.LoadUser(authedUserID); err == nil {
		return fmt.Errorf("user is already connected")
	}

	enc, ok := app.Remote.(remote.CaldavPasswordEncoder)
	if !ok {
		return errors.New("this calendar provider does not support app-password connect")
	}
	tok := &oauth2.Token{
		AccessToken: enc.EncodeCalDAVCredentials(email, appPassword),
		Expiry:      time.Date(caldavTokenExpiry, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	ctx := context.Background()
	client := app.Remote.MakeUserClient(ctx, tok, authedUserID, app.Poster, app.Store)
	me, err := client.GetMe()
	if err != nil {
		return err
	}

	uid, err := app.Store.LoadMattermostUserID(me.ID)
	if err == nil {
		user, userErr := app.PluginAPI.GetMattermostUser(uid)
		if userErr == nil {
			msg := app.Tr(authedUserID, "ycal.oauth.remote_already_connected",
				"{{.DisplayName}} account `{{.Mail}}` is already mapped to Mattermost account `{{.Username}}`. Please run `/{{.Trigger}} disconnect`, while logged in as the Mattermost account",
				map[string]any{"DisplayName": config.Provider.DisplayName, "Mail": me.Mail, "Username": user.Username, "Trigger": config.Provider.CommandTrigger})
			app.Poster.DM(authedUserID, msg)
			return errors.New(msg)
		}

		if userErr == store.ErrNotFound {
			msg := app.Tr(authedUserID, "ycal.oauth.remote_already_disabled",
				"{{.DisplayName}} account `{{.Mail}}` is already mapped to a Mattermost account, but the account is deactivated. Please enable it and run `/{{.Trigger}} disconnect`,  while logged in as the other Mattermost account, and try again",
				map[string]any{"DisplayName": config.Provider.DisplayName, "Mail": me.Mail, "Trigger": config.Provider.CommandTrigger})
			app.Poster.DM(authedUserID, msg)
			return errors.New(msg)
		}

		msg := app.Tr(authedUserID, "ycal.oauth.remote_already_not_found",
			"{{.DisplayName}} account `{{.Mail}}` is already mapped to a Mattermost account, but the Mattermost user could not be found",
			map[string]any{"DisplayName": config.Provider.DisplayName, "Mail": me.Mail})
		app.Poster.DM(authedUserID, msg)
		return errors.New(msg)
	}

	mmUser, userErr := app.PluginAPI.GetMattermostUser(authedUserID)
	if userErr != nil {
		return fmt.Errorf("error retrieving mattermost user (%s): %w", authedUserID, userErr)
	}

	u := &store.User{
		PluginVersion:         app.Config.PluginVersion,
		MattermostUserID:      authedUserID,
		MattermostUsername:    mmUser.Username,
		MattermostDisplayName: mmUser.GetDisplayName(model.ShowFullName),
		Remote:                me,
		OAuth2Token:           tok,
		Settings:              store.DefaultSettings,
	}

	mailboxSettings, err := client.GetMailboxSettings(me.ID)
	if err != nil {
		return err
	}

	u.Settings.DailySummary = &store.DailySummaryUserSettings{
		PostTime: "8:00AM",
		Timezone: mailboxSettings.TimeZone,
		Enable:   false,
	}

	err = app.Store.StoreUser(u)
	if err != nil {
		return err
	}

	err = app.Store.StoreUserInIndex(u)
	if err != nil {
		return err
	}

	// Seed stored events so the first poll does not DM every event as "new".
	if seedErr := app.seedUserEventsFromClient(authedUserID, client); seedErr != nil {
		app.Logger.Warnf("CalDAV connect: could not seed user events: %v", seedErr)
	}

	app.Welcomer.AfterSuccessfullyConnect(authedUserID, me.Mail)

	return nil
}

func (app *oauth2App) seedUserEventsFromClient(mattermostUserID string, client remoteCalendarClient) error {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now().Add(60 * 24 * time.Hour)
	events, err := client.GetDefaultCalendarView("", start, end)
	if err != nil {
		return err
	}
	for _, ev := range events {
		if ev == nil || ev.ICalUID == "" {
			continue
		}
		se := &store.Event{
			Remote:        ev,
			PluginVersion: app.Config.PluginVersion,
		}
		if err := app.Store.StoreUserEvent(mattermostUserID, se); err != nil {
			app.Logger.Warnf("seed event %s: %v", ev.ICalUID, err)
		}
	}
	return nil
}

// remoteCalendarClient is the subset of remote.Client needed for seeding.
type remoteCalendarClient interface {
	GetDefaultCalendarView(remoteUserID string, startTime, endTime time.Time) ([]*remote.Event, error)
}
