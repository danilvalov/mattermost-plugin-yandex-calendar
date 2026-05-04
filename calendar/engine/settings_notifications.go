// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/settingspanel"
)

type notificationSetting struct {
	getCal      func(string) Engine
	title       string
	description string
	id          string
	dependsOn   string
	tr          settingspanel.Translator
}

func NewNotificationsSetting(getCal func(string) Engine, tr settingspanel.Translator) settingspanel.Setting {
	return &notificationSetting{
		title:       "Receive notifications of new events",
		description: "Do you want to subscribe to new events and receive a message when they are created?",
		id:          "new_or_updated_event_setting",
		dependsOn:   "",
		getCal:      getCal,
		tr:          tr,
	}
}

func (s *notificationSetting) Set(userID string, value interface{}) error {
	boolValue := false
	if value == "true" {
		boolValue = true
	}

	cal := s.getCal(userID)

	if boolValue {
		_, err := cal.LoadMyEventSubscription()
		if err != nil {
			_, err := cal.CreateMyEventSubscription()
			if err != nil {
				return err
			}
		}

		return nil
	}

	_, err := cal.LoadMyEventSubscription()
	if err == nil {
		return cal.DeleteMyEventSubscription()
	}
	return nil
}

func (s *notificationSetting) Get(userID string) (interface{}, error) {
	cal := s.getCal(userID)
	_, err := cal.LoadMyEventSubscription()
	if err == nil {
		return "true", nil
	}

	return "false", nil
}

func (s *notificationSetting) GetID() string {
	return s.id
}

func (s *notificationSetting) GetTitle() string {
	return s.title
}

func (s *notificationSetting) GetDescription() string {
	return s.description
}

func (s *notificationSetting) GetDependency() string {
	return s.dependsOn
}

func (s *notificationSetting) getActionStyle(actionValue, currentValue string) string {
	if actionValue == currentValue {
		return "primary"
	}
	return "default"
}

func (s *notificationSetting) GetSlackAttachments(userID, settingHandler string, disabled bool) (*model.SlackAttachment, error) {
	key := "ycal.settings." + s.id + "."
	locTitle := s.tr.T(userID, key+"title", s.title, nil)
	locDesc := s.tr.T(userID, key+"desc", s.description, nil)
	title := s.tr.T(userID, "ycal.settings.ui.attachment_title", "Setting: {{.Name}}", map[string]any{"Name": locTitle})
	currentValueMessage := s.tr.T(userID, "ycal.settings.ui.disabled", "Disabled", nil)

	actions := []*model.PostAction{}
	if !disabled {
		currentValue, err := s.Get(userID)
		if err != nil {
			return nil, err
		}

		curStr, _ := currentValue.(string)
		currentTextValue := s.tr.T(userID, "ycal.settings.ui.no", "No", nil)
		if curStr == "true" {
			currentTextValue = s.tr.T(userID, "ycal.settings.ui.yes", "Yes", nil)
		}
		currentValueMessage = s.tr.T(userID, "ycal.settings.ui.current_value", "**Current value:** {{.Value}}", map[string]any{"Value": currentTextValue})

		yes := s.tr.T(userID, "ycal.settings.ui.yes", "Yes", nil)
		no := s.tr.T(userID, "ycal.settings.ui.no", "No", nil)

		actionTrue := model.PostAction{
			Name:  yes,
			Style: s.getActionStyle("true", curStr),
			Integration: &model.PostActionIntegration{
				URL: settingHandler,
				Context: map[string]interface{}{
					settingspanel.ContextIDKey:          s.id,
					settingspanel.ContextButtonValueKey: "true",
				},
			},
		}

		actionFalse := model.PostAction{
			Name:  no,
			Style: s.getActionStyle("false", curStr),
			Integration: &model.PostActionIntegration{
				URL: settingHandler,
				Context: map[string]interface{}{
					settingspanel.ContextIDKey:          s.id,
					settingspanel.ContextButtonValueKey: "false",
				},
			},
		}
		actions = []*model.PostAction{&actionTrue, &actionFalse}
	}

	text := fmt.Sprintf("%s\n%s", locDesc, currentValueMessage)
	sa := model.SlackAttachment{
		Title:    title,
		Text:     text,
		Actions:  actions,
		Fallback: fmt.Sprintf("%s: %s", title, text),
	}

	return &sa, nil
}

func (s *notificationSetting) IsDisabled(foreignValue interface{}) bool {
	return foreignValue == "false"
}
