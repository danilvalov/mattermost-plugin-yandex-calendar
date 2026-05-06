// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
)

type Users interface {
	GetActingUser() *User
	GetTimezone(user *User) (string, error)
	IsMilitaryTime(user *User) bool
	DisconnectUser(mattermostUserID string) error
	GetRemoteUser(mattermostUserID string) (*remote.User, error)
	IsAuthorizedAdmin(mattermostUserID string) (bool, error)
	GetUserSettings(user *User) (*store.Settings, error)
}

type User struct {
	*store.User
	MattermostUser   *model.User
	MattermostUserID string
}

func NewUser(mattermostUserID string) *User {
	return &User{
		MattermostUserID: mattermostUserID,
	}
}

func newUserFromStoredUser(u *store.User) *User {
	return &User{
		User:             u,
		MattermostUserID: u.MattermostUserID,
	}
}

func (user *User) Clone() *User {
	clone := *user
	clone.User = user.User.Clone()
	return &clone
}

func (m *mscalendar) GetActingUser() *User {
	return m.actingUser
}

func (m *mscalendar) ExpandUser(user *User) error {
	err := m.ExpandRemoteUser(user)
	if err != nil {
		return err
	}
	err = m.ExpandMattermostUser(user)
	if err != nil {
		return err
	}
	return nil
}

func (m *mscalendar) ExpandRemoteUser(user *User) error {
	if user.User == nil {
		storedUser, err := m.Store.LoadUser(user.MattermostUserID)
		if err != nil {
			return errors.Wrapf(err, "It looks like your Mattermost account is not connected to %s. Please connect your account using `/%s connect`.", m.Provider.DisplayName, m.Provider.CommandTrigger) //nolint:revive
		}
		user.User = storedUser
	}
	return nil
}

func (m *mscalendar) ExpandMattermostUser(user *User) error {
	if user.MattermostUser == nil {
		mattermostUser, err := m.PluginAPI.GetMattermostUser(user.MattermostUserID)
		if err != nil {
			return err
		}
		user.MattermostUser = mattermostUser
	}
	return nil
}

func (m *mscalendar) GetTimezone(user *User) (string, error) {
	err := m.Filter(
		withClient,
		withUserExpanded(user),
	)
	if err != nil {
		return "", err
	}

	if user.MattermostUser != nil && user.MattermostUser.Timezone != nil {
		if user.MattermostUser.Timezone["useAutomaticTimezone"] == "true" {
			if timezone := user.MattermostUser.Timezone["automaticTimezone"]; timezone != "" {
				return timezone, nil
			}
		}
		if timezone := user.MattermostUser.Timezone["manualTimezone"]; timezone != "" {
			return timezone, nil
		}
	}

	settings, err := m.client.GetMailboxSettings(user.Remote.ID)
	if err != nil {
		return "", err
	}
	if settings.TimeZone != "" {
		return settings.TimeZone, nil
	}

	return "UTC", nil
}

func (m *mscalendar) GetTimezoneByID(mattermostUserID string) (string, error) {
	return m.GetTimezone(NewUser(mattermostUserID))
}

func (m *mscalendar) IsMilitaryTime(user *User) bool {
	pref, err := m.PluginAPI.GetPreferenceForUser(user.MattermostUserID, preferenceCategoryDisplay, preferenceUseMilitaryTime)
	if err != nil || pref == nil {
		pref, err = m.PluginAPI.GetPreferenceForUser(user.MattermostUserID, preferenceCategoryDisplaySettings, preferenceUseMilitaryTime)
	}
	if err != nil || pref == nil {
		return m.isMilitaryTimeFromAllPreferences(user.MattermostUserID)
	}
	return pref.Value == "true"
}

func (m *mscalendar) isMilitaryTimeFromAllPreferences(mattermostUserID string) bool {
	prefs, err := m.PluginAPI.GetPreferencesForUser(mattermostUserID)
	if err != nil {
		return false
	}

	militaryNames := map[string]struct{}{
		"use_military_time": {},
		"useMilitaryTime":   {},
		"military_time":     {},
	}

	for _, pref := range prefs {
		if pref.Category == preferenceCategoryDisplay || pref.Category == preferenceCategoryDisplaySettings {
			if _, ok := militaryNames[pref.Name]; ok {
				return pref.Value == "true"
			}
		}
	}
	return false
}

func (user *User) String() string {
	if user.MattermostUser != nil {
		return fmt.Sprintf("@%s", user.MattermostUser.Username)
	}

	return user.MattermostUserID
}

func (user *User) Markdown() string {
	if user.MattermostUser != nil {
		return fmt.Sprintf("@%s", user.MattermostUser.Username)
	}

	return fmt.Sprintf("UserID: `%s`", user.MattermostUserID)
}

func (m *mscalendar) DisconnectUser(mattermostUserID string) error {
	m.AfterDisconnect(mattermostUserID)
	err := m.Filter(
		withClient,
	)
	if err != nil {
		return err
	}

	storedUser, err := m.Store.LoadUser(mattermostUserID)
	if err != nil {
		if err == store.ErrNotFound {
			return err
		}

		// Fall back to force-deleting using the unencrypted user index.
		m.Logger.Warnf("failed to load user %s during disconnect, attempting force-delete: %v", mattermostUserID, err)
		indexUser, indexErr := m.Store.LoadUserFromIndex(mattermostUserID)
		if indexErr != nil {
			return errors.Wrapf(err, "unable to load user for disconnect (index lookup also failed: %v)", indexErr)
		}
		return m.Store.ForceDeleteUser(mattermostUserID, indexUser.RemoteID)
	}

	// Unlink events owned by the user that is disconnecting its account
	linkedEventsLeft := make(map[string]string)
	for eventID, channelID := range storedUser.ChannelEvents {
		if errStore := m.Store.DeleteLinkedChannelFromEvent(eventID, channelID); errStore != nil {
			linkedEventsLeft[eventID] = channelID
		}
	}
	if len(linkedEventsLeft) != 0 {
		storedUser.ChannelEvents = linkedEventsLeft
		if errStore := m.Store.StoreUser(storedUser); errStore != nil {
			m.Logger.With(bot.LogContext{
				"err":                  errStore,
				"mm_user_id":           storedUser.MattermostDisplayName,
				"linked_channels_left": linkedEventsLeft,
			}).Errorf("error storing user after failing deleting linked channels from store")
		}
		return fmt.Errorf("error deleting linked channels from events")
	}

	eventSubscriptionID := storedUser.Settings.EventSubscriptionID
	if eventSubscriptionID != "" {
		sub, errLoad := m.Store.LoadSubscription(eventSubscriptionID)
		if errLoad != nil {
			return errors.Wrap(errLoad, "error loading subscription")
		}

		err = m.Store.DeleteUserSubscription(storedUser, eventSubscriptionID)
		if err != nil && err != store.ErrNotFound {
			return errors.WithMessagef(err, "failed to delete subscription %s", eventSubscriptionID)
		}

		err = m.client.DeleteSubscription(sub.Remote)
		if err != nil {
			m.Logger.Warnf("failed to delete remote subscription %s. err=%v", eventSubscriptionID, err)
		}
	}

	err = m.Store.DeleteUser(mattermostUserID)
	if err != nil {
		return err
	}

	err = m.Store.DeleteUserFromIndex(mattermostUserID)
	if err != nil {
		return err
	}

	return nil
}

func (m *mscalendar) GetRemoteUser(mattermostUserID string) (*remote.User, error) {
	storedUser, err := m.Store.LoadUser(mattermostUserID)
	if err != nil {
		return nil, err
	}

	return storedUser.Remote, nil
}

func (m *mscalendar) IsAuthorizedAdmin(mattermostUserID string) (bool, error) {
	for _, userID := range strings.Split(m.AdminUserIDs, ",") {
		if userID == mattermostUserID {
			return true, nil
		}
	}

	ok, err := m.PluginAPI.IsSysAdmin(mattermostUserID)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (m *mscalendar) GetUserSettings(user *User) (*store.Settings, error) {
	err := m.Filter(
		withUserExpanded(user),
	)
	if err != nil {
		return nil, err
	}

	return &user.Settings, nil
}
