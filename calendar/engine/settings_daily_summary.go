// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/settingspanel"
)

type dailySummarySetting struct {
	store       settingspanel.SettingStore
	getTimezone func(userID string) (string, error)
	title       string
	dependsOn   string
	description string
	id          string
	optionsH    []string
	optionsM    []string
	optionsAPM  []string
	tr          settingspanel.Translator
}

func NewDailySummarySetting(inStore settingspanel.SettingStore, getTimezone func(userID string) (string, error), tr settingspanel.Translator) settingspanel.Setting {
	os := &dailySummarySetting{
		title:       "Daily Summary",
		description: "When do you want to receive the daily summary?\n If you update this setting, it will automatically update to your the timezone currently set on your calendar.",
		id:          store.DailySummarySettingID,
		dependsOn:   "",
		store:       inStore,
		getTimezone: getTimezone,
		tr:          tr,
	}
	os.optionsH = []string{"12"}
	for i := 1; i < 12; i++ {
		os.optionsH = append(os.optionsH, fmt.Sprintf("%d", i))
	}

	os.optionsM = []string{}
	for i := 0; i < 4; i++ {
		os.optionsM = append(os.optionsM, fmt.Sprintf("%02d", i*15))
	}

	os.optionsAPM = []string{"AM", "PM"}

	return os
}

func (s *dailySummarySetting) Set(userID string, value interface{}) error {
	_, ok := value.(string)
	if !ok {
		return errors.New("trying to set Daily Summary Setting without a string value")
	}
	err := s.store.SetSetting(userID, s.id, value)
	if err != nil {
		return err
	}

	return nil
}

func (s *dailySummarySetting) Get(userID string) (interface{}, error) {
	value, err := s.store.GetSetting(userID, s.id)
	if err != nil {
		return nil, err
	}

	_, ok := value.(*store.DailySummaryUserSettings)
	if !ok {
		return nil, errors.New("current value is not a Daily Summary Setting")
	}

	return value, nil
}

func (s *dailySummarySetting) GetID() string {
	return s.id
}

func (s *dailySummarySetting) GetTitle() string {
	return s.title
}

func (s *dailySummarySetting) GetDescription() string {
	return s.description
}

func (s *dailySummarySetting) GetDependency() string {
	return s.dependsOn
}

func (s *dailySummarySetting) GetSlackAttachments(userID, settingHandler string, disabled bool) (*model.SlackAttachment, error) {
	key := "ycal.settings." + s.id + "."
	locTitle := s.tr.T(userID, key+"title", s.title, nil)
	locDesc := s.tr.T(userID, key+"desc", s.description, nil)
	title := s.tr.T(userID, "ycal.settings.ui.attachment_title", "Setting: {{.Name}}", map[string]any{"Name": locTitle})
	currentValueMessage := s.tr.T(userID, "ycal.settings.ui.disabled", "Disabled", nil)

	actions := []*model.PostAction{}

	if disabled {
		text := fmt.Sprintf("%s\n%s", locDesc, currentValueMessage)
		sa := model.SlackAttachment{
			Title:    title,
			Text:     text,
			Actions:  actions,
			Fallback: fmt.Sprintf("%s: %s", title, text),
		}
		return &sa, nil
	}

	dsumRaw, err := s.Get(userID)
	if err != nil {
		return nil, err
	}
	dsum := dsumRaw.(*store.DailySummaryUserSettings)

	currentH := "8"
	currentM := "00"
	currentAPM := "AM"
	fullTime := "8:00AM"
	currentEnable := false

	if dsum != nil {
		fullTime = dsum.PostTime
		currentEnable = dsum.Enable
		splitted := strings.Split(fullTime, ":")
		currentH = splitted[0]
		currentM = splitted[1][:2]
		currentAPM = splitted[1][2:]
	}

	timezone, err := s.getTimezone(userID)
	if err != nil {
		return nil, fmt.Errorf("could not load the timezone. err=%v", err)
	}
	fullTime = fullTime + " " + timezone

	actionOptionsH := model.PostAction{
		Name: s.tr.T(userID, "ycal.settings.daily.label_hour", "H:", nil),
		Integration: &model.PostActionIntegration{
			URL: settingHandler,
			Context: map[string]interface{}{
				settingspanel.ContextIDKey: s.id,
			},
		},
		Type:          "select",
		Options:       s.makeHOptions(currentM, currentAPM, timezone),
		DefaultOption: fullTime,
	}

	actionOptionsM := model.PostAction{
		Name: s.tr.T(userID, "ycal.settings.daily.label_minute", "M:", nil),
		Integration: &model.PostActionIntegration{
			URL: settingHandler,
			Context: map[string]interface{}{
				settingspanel.ContextIDKey: s.id,
			},
		},
		Type:          "select",
		Options:       s.makeMOptions(currentH, currentAPM, timezone),
		DefaultOption: fullTime,
	}

	actionOptionsAPM := model.PostAction{
		Name: s.tr.T(userID, "ycal.settings.daily.label_ampm", "AM/PM:", nil),
		Integration: &model.PostActionIntegration{
			URL: settingHandler,
			Context: map[string]interface{}{
				settingspanel.ContextIDKey: s.id,
			},
		},
		Type:          "select",
		Options:       s.makeAPMOptions(userID, currentH, currentM, timezone),
		DefaultOption: fullTime,
	}

	if currentEnable {
		actions = []*model.PostAction{&actionOptionsH, &actionOptionsM, &actionOptionsAPM}
	}

	buttonText := s.tr.T(userID, "ycal.settings.daily.enable", "Enable", nil)
	enable := "true"
	if currentEnable {
		buttonText = s.tr.T(userID, "ycal.settings.daily.disable", "Disable", nil)
		enable = "false"
	}
	actionToggle := model.PostAction{
		Name: buttonText,
		Integration: &model.PostActionIntegration{
			URL: settingHandler,
			Context: map[string]interface{}{
				settingspanel.ContextIDKey:          s.id,
				settingspanel.ContextButtonValueKey: enable + " " + timezone,
			},
		},
	}

	actions = append(actions, &actionToggle)

	sa := model.SlackAttachment{
		Title:    title,
		Text:     locDesc,
		Actions:  actions,
		Fallback: fmt.Sprintf("%s: %s", title, locDesc),
	}
	return &sa, nil
}

func (s *dailySummarySetting) IsDisabled(foreignValue interface{}) bool {
	return foreignValue == "false"
}

func (s *dailySummarySetting) makeHOptions(minute, apm, timezone string) []*model.PostActionOptions {
	out := []*model.PostActionOptions{}
	for _, o := range s.optionsH {
		out = append(out, &model.PostActionOptions{
			Text:  o,
			Value: fmt.Sprintf("%s:%s%s %s", o, minute, apm, timezone),
		})
	}
	return out
}

func (s *dailySummarySetting) makeMOptions(hour, apm, timezone string) []*model.PostActionOptions {
	out := []*model.PostActionOptions{}
	for _, o := range s.optionsM {
		out = append(out, &model.PostActionOptions{
			Text:  o,
			Value: fmt.Sprintf("%s:%s%s %s", hour, o, apm, timezone),
		})
	}
	return out
}

func (s *dailySummarySetting) makeAPMOptions(userID, hour, minute, timezone string) []*model.PostActionOptions {
	out := []*model.PostActionOptions{}

	for _, o := range s.optionsAPM {
		display := o
		switch o {
		case "AM":
			display = s.tr.T(userID, "ycal.settings.daily.am", o, nil)
		case "PM":
			display = s.tr.T(userID, "ycal.settings.daily.pm", o, nil)
		}
		out = append(out, &model.PostActionOptions{
			Text:  display,
			Value: fmt.Sprintf("%s:%s%s %s", hour, minute, o, timezone),
		})
	}

	return out
}
