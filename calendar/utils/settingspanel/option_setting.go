// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package settingspanel

import (
	"errors"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

type optionSetting struct {
	store         SettingStore
	title         string
	description   string
	id            string
	dependsOn     string
	defaultOption string
	options       []string
	tr            Translator
}

func NewOptionSetting(id, title, description, dependsOn, defaultOption string, options []string, store SettingStore, tr Translator) Setting {
	return &optionSetting{
		title:         title,
		description:   description,
		id:            id,
		dependsOn:     dependsOn,
		options:       options,
		store:         store,
		defaultOption: defaultOption,
		tr:            tr,
	}
}

func (s *optionSetting) Set(userID string, value interface{}) error {
	err := s.store.SetSetting(userID, s.id, value)
	if err != nil {
		return err
	}

	return nil
}

func (s *optionSetting) Get(userID string) (interface{}, error) {
	value, err := s.store.GetSetting(userID, s.id)
	if err != nil {
		return "", err
	}
	valueString, ok := value.(string)
	if !ok {
		return "", errors.New("current value is not a string")
	}

	return valueString, nil
}

func (s *optionSetting) GetID() string {
	return s.id
}

func (s *optionSetting) GetTitle() string {
	return s.title
}

func (s *optionSetting) GetDescription() string {
	return s.description
}

func (s *optionSetting) GetDependency() string {
	return s.dependsOn
}

func (s *optionSetting) optionDisplayLabel(userID, value string) string {
	switch value {
	case "Away":
		return s.tr.T(userID, "ycal.settings.status.opt.away", value, nil)
	case "Do Not Disturb":
		return s.tr.T(userID, "ycal.settings.status.opt.dnd", value, nil)
	case "Don't set status for me":
		return s.tr.T(userID, "ycal.settings.status.opt.notset", value, nil)
	default:
		return value
	}
}

func (s *optionSetting) stringsToLocalizedOptions(userID string) []*model.PostActionOptions {
	out := []*model.PostActionOptions{}
	for _, o := range s.options {
		out = append(out, &model.PostActionOptions{
			Text:  s.optionDisplayLabel(userID, o),
			Value: o,
		})
	}
	return out
}

func (s *optionSetting) GetSlackAttachments(userID, settingHandler string, disabled bool) (*model.SlackAttachment, error) {
	key := "ycal.settings." + s.id + "."
	locTitle := s.tr.T(userID, key+"title", s.title, nil)
	locDesc := s.tr.T(userID, key+"desc", s.description, nil)
	title := s.tr.T(userID, "ycal.settings.ui.attachment_title", "Setting: {{.Name}}", map[string]any{"Name": locTitle})
	currentValueMessage := s.tr.T(userID, "ycal.settings.ui.disabled", "Disabled", nil)

	actions := []*model.PostAction{}
	if !disabled {
		currentTextValue, err := s.Get(userID)
		if err != nil {
			return nil, err
		}

		if currentTextValue == "" {
			currentTextValue = s.defaultOption
		}

		curStr, _ := currentTextValue.(string)
		displayVal := s.optionDisplayLabel(userID, curStr)
		currentValueMessage = s.tr.T(userID, "ycal.settings.ui.current_value", "**Current value:** {{.Value}}", map[string]any{"Value": displayVal})

		actionOptions := model.PostAction{
			Name: s.tr.T(userID, "ycal.settings.ui.select_option", "Select an option:", nil),
			Integration: &model.PostActionIntegration{
				URL: settingHandler + "?" + s.id + "=true",
				Context: map[string]interface{}{
					ContextIDKey: s.id,
				},
			},
			Type:    "select",
			Options: s.stringsToLocalizedOptions(userID),
		}

		actions = []*model.PostAction{&actionOptions}
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

func (s *optionSetting) IsDisabled(foreignValue interface{}) bool {
	return foreignValue == "false"
}
