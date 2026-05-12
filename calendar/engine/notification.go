// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
)

const maxQueueSize = 1024
const notificationDedupeTTL = 2 * time.Minute
const (
	preferenceCategoryDisplay         = "display"
	preferenceCategoryDisplaySettings = "display_settings"
	preferenceUseMilitaryTime         = "use_military_time"
)

const (
	FieldSubject        = "Subject"
	FieldBodyPreview    = "BodyPreview"
	FieldImportance     = "Importance"
	FieldDuration       = "Duration"
	FieldWhen           = "When"
	FieldLocation       = "Location"
	FieldAttendees      = "Attendees"
	FieldOrganizer      = "Organizer"
	FieldResponseStatus = "ResponseStatus"
)

const (
	OptionYes          = "Yes"
	OptionNotResponded = "Not responded"
	OptionNo           = "No"
	OptionMaybe        = "Maybe"
)

const (
	ResponseYes   = "accepted"
	ResponseMaybe = "tentativelyAccepted"
	ResponseNo    = "declined"
	ResponseNone  = "notResponded"
)

var importantNotificationChanges = []string{FieldSubject, FieldWhen}

var notificationFieldOrder = []string{
	FieldWhen,
	FieldLocation,
	FieldAttendees,
	FieldImportance,
}

type NotificationProcessor interface {
	Configure(Env)
	Enqueue(notifications ...*remote.Notification) error
	Quit()
}

type notificationProcessor struct {
	Env
	envChan chan Env

	queue chan *remote.Notification
	quit  chan bool
}

func NewNotificationProcessor(env Env) NotificationProcessor {
	processor := &notificationProcessor{
		Env:     env,
		envChan: make(chan (Env)),
		queue:   make(chan (*remote.Notification), maxQueueSize),
		quit:    make(chan (bool)),
	}
	go processor.work()
	return processor
}

func (processor *notificationProcessor) Enqueue(notifications ...*remote.Notification) error {
	for _, n := range notifications {
		select {
		case processor.queue <- n:
		default:
			return fmt.Errorf("webhook notification: queue full, dropped notification")
		}
	}
	return nil
}

func (processor *notificationProcessor) Configure(env Env) {
	processor.envChan <- env
}

func (processor *notificationProcessor) Quit() {
	processor.quit <- true
}

func (processor *notificationProcessor) work() {
	for {
		select {
		case n := <-processor.queue:
			err := processor.processNotification(n)
			if err != nil {
				processor.Logger.With(bot.LogContext{
					"subscriptionID": n.SubscriptionID,
				}).Infof("webhook notification: failed: `%v`.", err)
			}

		case env := <-processor.envChan:
			processor.Env = env

		case <-processor.quit:
			return
		}
	}
}

func (processor *notificationProcessor) processNotification(n *remote.Notification) error {
	sub, err := processor.Store.LoadSubscription(n.SubscriptionID)
	if err != nil {
		return err
	}
	creator, err := processor.Store.LoadUser(sub.MattermostCreatorID)
	if err != nil {
		return err
	}
	if sub.Remote.ID != creator.Settings.EventSubscriptionID {
		return errors.New("subscription is orphaned")
	}
	if sub.Remote.ClientState != "" && sub.Remote.ClientState != n.ClientState {
		return errors.New("unauthorized webhook")
	}

	n.Subscription = sub.Remote
	n.SubscriptionCreator = creator.Remote

	client := processor.Remote.MakeUserClient(context.Background(), creator.OAuth2Token, sub.MattermostCreatorID, processor.Poster, processor.Store)

	if n.RecommendRenew {
		var renewed *remote.Subscription
		renewed, err = client.RenewSubscription(processor.Config.GetNotificationURL(), sub.Remote.CreatorID, n.Subscription)
		if err != nil {
			return err
		}

		storedSub := &store.Subscription{
			Remote:              renewed,
			MattermostCreatorID: creator.MattermostUserID,
			PluginVersion:       processor.Config.PluginVersion,
		}
		err = processor.Store.StoreUserSubscription(creator, storedSub)
		if err != nil {
			return err
		}
		processor.Logger.With(bot.LogContext{
			"MattermostUserID": creator.MattermostUserID,
			"SubscriptionID":   n.SubscriptionID,
		}).Debugf("webhook notification: renewed user subscription.")
	}

	if n.IsBare {
		n, err = client.GetNotificationData(n)
		if err != nil {
			return err
		}
	}
	if n == nil || n.Event == nil || n.Event.ICalUID == "" {
		processor.Logger.Warnf("webhook notification: skipped invalid event payload")
		return nil
	}

	dedupeKey := notificationDedupeKey(creator.MattermostUserID, n)

	reserved, err := processor.Store.TryReserveNotification(dedupeKey, notificationDedupeTTL)
	if err != nil {
		return err
	}
	if !reserved {
		return nil
	}

	var sa *model.SlackAttachment
	prior, err := processor.Store.LoadUserEvent(creator.MattermostUserID, store.UserEventSnapshotKey(n.Event))
	if err != nil && err != store.ErrNotFound {
		return err
	}

	timezone, isMilitary := processor.getUserTimeInfo(creator.MattermostUserID)

	if prior != nil {
		var changed bool
		changed, sa = processor.updatedEventSlackAttachment(creator.MattermostUserID, n, prior.Remote, timezone, isMilitary)
		if !changed {
			return nil
		}
	} else {
		sa = processor.newEventSlackAttachment(creator.MattermostUserID, n, timezone, isMilitary)
		prior = &store.Event{}
	}

	_, err = processor.Poster.DMWithAttachments(creator.MattermostUserID, sa)
	if err != nil {
		return err
	}

	prior.Remote = n.Event
	err = processor.Store.StoreUserEvent(creator.MattermostUserID, prior)
	if err != nil {
		return err
	}

	return nil
}

func notificationDedupeSignature(n *remote.Notification) string {
	if n == nil || n.Event == nil {
		return ""
	}
	ev := n.Event
	start := ""
	if ev.Start != nil {
		start = ev.Start.String()
	}
	end := ""
	if ev.End != nil {
		end = ev.End.String()
	}
	location := ""
	if ev.Location != nil {
		location = ev.Location.DisplayName
	}

	return fmt.Sprintf("%s|%s|%s|%s|%s|%t|%s",
		n.ChangeType,
		ev.ICalUID,
		ev.Subject,
		start,
		end,
		ev.IsCancelled,
		location,
	)
}

func notificationDedupeKey(mattermostUserID string, n *remote.Notification) string {
	sig := notificationDedupeSignature(n)
	sum := sha256.Sum256([]byte(sig))
	return mattermostUserID + "_" + hex.EncodeToString(sum[:])
}

func (processor *notificationProcessor) getUserTimeInfo(mattermostUserID string) (timezone string, isMilitary bool) {
	timezone = "UTC"

	user, err := processor.PluginAPI.GetMattermostUser(mattermostUserID)
	if err == nil && user != nil && user.Timezone != nil {
		if user.Timezone["useAutomaticTimezone"] == "true" {
			timezone = user.Timezone["automaticTimezone"]
		} else if user.Timezone["manualTimezone"] != "" {
			timezone = user.Timezone["manualTimezone"]
		}
	}

	pref, err := processor.PluginAPI.GetPreferenceForUser(mattermostUserID, preferenceCategoryDisplay, preferenceUseMilitaryTime)
	if err != nil || pref == nil {
		pref, err = processor.PluginAPI.GetPreferenceForUser(mattermostUserID, preferenceCategoryDisplaySettings, preferenceUseMilitaryTime)
	}
	if err != nil {
		prefs, allErr := processor.PluginAPI.GetPreferencesForUser(mattermostUserID)
		if allErr == nil {
			for _, p := range prefs {
				if (p.Category == preferenceCategoryDisplay || p.Category == preferenceCategoryDisplaySettings) && p.Name == preferenceUseMilitaryTime {
					isMilitary = p.Value == "true"
					break
				}
			}
		}
	} else if pref != nil {
		isMilitary = pref.Value == "true"
	}

	return timezone, isMilitary
}
