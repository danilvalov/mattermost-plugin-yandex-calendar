// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package settingspanel

import (
	"errors"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

type boolSetting struct {
	store       SettingStore
	title       string
	description string
	id          string
	dependsOn   string
	tr          Translator
}

func NewBoolSetting(id string, title string, description string, dependsOn string, store SettingStore, tr Translator) Setting {
	return &boolSetting{
		title:       title,
		description: description,
		id:          id,
		dependsOn:   dependsOn,
		store:       store,
		tr:          tr,
	}
}

func (s *boolSetting) Set(userID string, value interface{}) error {
	boolValue := false
	if value == "true" {
		boolValue = true
	}

	err := s.store.SetSetting(userID, s.id, boolValue)
	if err != nil {
		return err
	}

	return nil
}

func (s *boolSetting) Get(userID string) (interface{}, error) {
	value, err := s.store.GetSetting(userID, s.id)
	if err != nil {
		return "", err
	}
	boolValue, ok := value.(bool)
	if !ok {
		return "", errors.New("current value is not a bool")
	}

	stringValue := "false"
	if boolValue {
		stringValue = "true"
	}

	return stringValue, nil
}

func (s *boolSetting) GetID() string {
	return s.id
}

func (s *boolSetting) GetTitle() string {
	return s.title
}

func (s *boolSetting) GetDescription() string {
	return s.description
}

func (s *boolSetting) GetDependency() string {
	return s.dependsOn
}

func (s *boolSetting) getActionStyle(actionValue, currentValue string) string {
	if actionValue == currentValue {
		return "primary"
	}
	return "default"
}

func (s *boolSetting) GetSlackAttachments(userID, settingHandler string, disabled bool) (*model.SlackAttachment, error) {
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
					ContextIDKey:          s.id,
					ContextButtonValueKey: "true",
				},
			},
		}

		actionFalse := model.PostAction{
			Name:  no,
			Style: s.getActionStyle("false", curStr),
			Integration: &model.PostActionIntegration{
				URL: settingHandler,
				Context: map[string]interface{}{
					ContextIDKey:          s.id,
					ContextButtonValueKey: "false",
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
		Fallback: fmt.Sprintf("%s: %s", title, locDesc),
	}

	return &sa, nil
}

func (s *boolSetting) IsDisabled(foreignValue interface{}) bool {
	return foreignValue == "false"
}
