// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/flow"
)

type Welcomer interface {
	Welcome(userID string) error
	AfterSuccessfullyConnect(userID, userLogin string) error
	AfterDisconnect(userID string) error
	WelcomeFlowEnd(userID string)
}

type Bot interface {
	bot.Bot
	Welcomer
	flow.Store
}

type mscBot struct {
	bot.Bot
	Env
	pluginURL string
}

func (m *mscalendar) Welcome(userID string) error {
	return m.Welcomer.Welcome(userID)
}

func (m *mscalendar) AfterSuccessfullyConnect(userID, userLogin string) error {
	return m.Welcomer.AfterSuccessfullyConnect(userID, userLogin)
}

func (m *mscalendar) AfterDisconnect(userID string) error {
	return m.Welcomer.AfterDisconnect(userID)
}

func (m *mscalendar) WelcomeFlowEnd(userID string) {
	m.Welcomer.WelcomeFlowEnd(userID)
}

func NewMSCalendarBot(bot bot.Bot, env Env, pluginURL string) Bot {
	return &mscBot{
		Bot:       bot,
		Env:       env,
		pluginURL: pluginURL,
	}
}

func (bot *mscBot) Welcome(userID string) error {
	bot.cleanWelcomePost(userID)

	postID, err := bot.DMWithAttachments(userID, bot.newConnectAttachment(userID))
	if err != nil {
		return err
	}

	bot.Store.StoreUserWelcomePost(userID, postID)

	return nil
}

func (bot *mscBot) AfterSuccessfullyConnect(userID, userLogin string) error {
	bot.PluginAPI.PublishWebsocketEvent(userID, "connected", map[string]any{"action": "connected"})

	bot.Tracker.TrackUserAuthenticated(userID)
	postID, err := bot.Store.DeleteUserWelcomePost(userID)
	if err != nil {
		bot.Errorf("error deleting user's welcome post id, err=%v", err)
	}
	if postID != "" {
		post := &model.Post{
			Id: postID,
		}
		model.ParseSlackAttachment(post, []*model.SlackAttachment{bot.newConnectedAttachment(userID, userLogin)})
		bot.UpdatePost(post)
	}

	return bot.Start(userID)
}

func (bot *mscBot) AfterDisconnect(userID string) error {
	bot.PluginAPI.PublishWebsocketEvent(userID, "disconnected", map[string]any{"action": "disconnected"})

	bot.Tracker.TrackUserDeauthenticated(userID)
	errCancel := bot.Cancel(userID)
	errClean := bot.cleanWelcomePost(userID)
	if errCancel != nil {
		return errCancel
	}

	if errClean != nil {
		return errClean
	}
	return nil
}

func (bot *mscBot) WelcomeFlowEnd(userID string) {
	bot.Tracker.TrackWelcomeFlowCompletion(userID)
	bot.notifySettings(userID)
}

func (bot *mscBot) newConnectAttachment(mattermostUserID string) *model.SlackAttachment {
	title := bot.Tr(mattermostUserID, "ycal.welcome.connect_title", "Connect", nil)
	var text string
	if bot.Provider.Features.PasswordAuth {
		text = bot.Tr(mattermostUserID, "ycal.welcome.password_auth",
			"Welcome to the {{.DisplayName}} plugin. [Open the connect page]({{.URL}}/caldav/connect) to enter your account email and app password.",
			map[string]any{"DisplayName": bot.Provider.DisplayName, "URL": bot.pluginURL})
	} else {
		text = bot.Tr(mattermostUserID, "ycal.welcome.oauth",
			"Welcome to the {{.DisplayName}} plugin. [Click here to link your account.]({{.URL}}/oauth2/connect)",
			map[string]any{"DisplayName": bot.Provider.DisplayName, "URL": bot.pluginURL})
	}
	sa := model.SlackAttachment{
		Title:    title,
		Text:     text,
		Fallback: fmt.Sprintf("%s: %s", title, text),
	}

	return &sa
}

func (bot *mscBot) newConnectedAttachment(mattermostUserID, userLogin string) *model.SlackAttachment {
	title := bot.Tr(mattermostUserID, "ycal.welcome.connect_title", "Connect", nil)
	text := bot.Tr(mattermostUserID, "ycal.welcome.connected",
		":tada: Congratulations! Your {{.DisplayName}} account (*{{.Login}}*) has been connected to Mattermost.",
		map[string]any{"DisplayName": bot.Provider.DisplayName, "Login": userLogin})
	return &model.SlackAttachment{
		Title:    title,
		Text:     text,
		Fallback: fmt.Sprintf("%s: %s", title, text),
	}
}

func (bot *mscBot) notifySettings(userID string) error {
	_, err := bot.DM(userID, bot.Tr(userID, "ycal.welcome.settings_hint",
		"Feel free to change these settings anytime by typing `/{{.Trigger}} settings`",
		map[string]any{"Trigger": config.Provider.CommandTrigger}))
	if err != nil {
		return err
	}
	return nil
}

func (bot *mscBot) cleanWelcomePost(mattermostUserID string) error {
	postID, err := bot.Store.DeleteUserWelcomePost(mattermostUserID)
	if err != nil {
		return err
	}

	if postID != "" {
		err = bot.DeletePost(postID)
		if err != nil {
			bot.Errorf("Error deleting post. err=%v", err)
		}
	}
	return nil
}

func (bot *mscBot) SetProperty(userID, propertyName string, value interface{}) error {
	if propertyName == store.SubscribePropertyName {
		if boolValue, _ := value.(bool); boolValue {
			m := New(bot.Env, userID)
			_, err := m.LoadMyEventSubscription()
			if err == nil { // Subscription found
				return nil
			}

			_, err = m.CreateMyEventSubscription()
			if err != nil {
				return err
			}
		}
		return nil
	}

	return bot.Dependencies.Store.SetProperty(userID, propertyName, value)
}

func (bot *mscBot) SetPostID(userID, propertyName, postID string) error {
	return bot.Dependencies.Store.SetPostID(userID, propertyName, postID)
}

func (bot *mscBot) GetPostID(userID, propertyName string) (string, error) {
	return bot.Dependencies.Store.GetPostID(userID, propertyName)
}

func (bot *mscBot) RemovePostID(userID, propertyName string) error {
	return bot.Dependencies.Store.RemovePostID(userID, propertyName)
}

func (bot *mscBot) GetCurrentStep(userID string) (int, error) {
	return bot.Dependencies.Store.GetCurrentStep(userID)
}
func (bot *mscBot) SetCurrentStep(userID string, step int) error {
	return bot.Dependencies.Store.SetCurrentStep(userID, step)
}
func (bot *mscBot) DeleteCurrentStep(userID string) error {
	return bot.Dependencies.Store.DeleteCurrentStep(userID)
}
