// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	mmi18n "github.com/mattermost/mattermost/server/public/pluginapi/i18n"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/locale"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/settingspanel"
)

type Settings interface {
	PrintSettings(userID string)
	ClearSettingsPosts(userID string)
}

func (m *mscalendar) PrintSettings(userID string) {
	m.SettingsPanel.Print(userID)
}

func (m *mscalendar) ClearSettingsPosts(userID string) {
	err := m.SettingsPanel.Clear(userID)
	if err != nil {
		m.Logger.Warnf("Error clearing settings posts. err=%v", err)
	}
}

// NewSettingsPanel builds the interactive settings UI. i18n should return the current plugin i18n
// bundle (may be nil before InitBundle); a getter is required so localization works if the panel
// was created before OnActivate initialized translations.
func NewSettingsPanel(bot bot.Bot, panelStore settingspanel.PanelStore, settingStore settingspanel.SettingStore, settingsHandler, pluginURL string, getCal func(userID string) Engine, providerFeatures config.ProviderFeatures, i18n func() *mmi18n.Bundle) settingspanel.Panel {
	tr := settingspanel.Translator(func(userID, id, def string, data map[string]any) string {
		var b *mmi18n.Bundle
		if i18n != nil {
			b = i18n()
		}
		return locale.User(b, userID, id, def, data)
	})
	settings := []settingspanel.Setting{}
	settings = append(settings, settingspanel.NewOptionSetting(
		store.UpdateStatusFromOptionsSettingID,
		"Update Status",
		"Do you want to update your status on Mattermost when you are in a meeting?",
		"",
		store.NotSetStatusOption,
		[]string{store.AwayStatusOption, store.DNDStatusOption, store.NotSetStatusOption},
		settingStore,
		tr,
	))
	settings = append(settings, settingspanel.NewBoolSetting(
		store.GetConfirmationSettingID,
		"Get Confirmation",
		"Do you want to get a confirmation before automatically updating your status?",
		store.UpdateStatusFromOptionsSettingID,
		settingStore,
		tr,
	))
	settings = append(settings, settingspanel.NewBoolSetting(
		store.SetCustomStatusSettingID,
		"Set Custom Status",
		"Do you want to set custom status automatically on Mattermost when you are in a meeting?",
		"",
		settingStore,
		tr,
	))
	settings = append(settings, settingspanel.NewBoolSetting(
		store.ReceiveRemindersSettingID,
		"Receive Reminders",
		"Do you want to receive reminders for upcoming events?",
		"",
		settingStore,
		tr,
	))
	if providerFeatures.EventNotifications {
		settings = append(settings, NewNotificationsSetting(getCal, tr))
	}
	settings = append(settings, NewDailySummarySetting(
		settingStore,
		func(userID string) (string, error) { return getCal(userID).GetTimezone(NewUser(userID)) },
		tr,
	))
	return settingspanel.NewSettingsPanel(settings, bot, bot, panelStore, settingsHandler, pluginURL)
}
