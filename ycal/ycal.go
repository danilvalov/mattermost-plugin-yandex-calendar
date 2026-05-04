// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package ycal

import "github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"

const (
	ProviderYCal            = "ycal"
	ProviderYCalDisplayName = "Yandex Calendar"
	ProviderYCalRepository  = "https://github.com/danilvalov/mattermost-plugin-yandex-calendar"
)

// GetYCalProviderConfig returns the Mattermost Calendar engine configuration for Yandex (CalDAV).
func GetYCalProviderConfig() config.ProviderConfig {
	return config.ProviderConfig{
		Name:        ProviderYCal,
		DisplayName: ProviderYCalDisplayName,
		Repository:  ProviderYCalRepository,

		CommandTrigger: ProviderYCal,

		TelemetryShortName: ProviderYCal,

		BotUsername:    ProviderYCal,
		BotDisplayName: ProviderYCalDisplayName,
		Features: config.ProviderFeatures{
			EncryptedStore:             true,
			EventNotifications:         true,
			ForceOAuth2Consent:         false,
			PasswordAuth:               true,
			EnableEventPolling:         true,
			HideCreateEventFromCommand: false,
		},
	}
}
