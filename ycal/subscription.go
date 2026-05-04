package ycal

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

const (
	subscriptionSuffix = "_ycal_poll_notifications"
	ycalClientState    = "ycal-poll-state"
)

func (c *client) CreateMySubscription(notificationURL, remoteUserID string) (*remote.Subscription, error) {
	sub := &remote.Subscription{
		ID:                 fmt.Sprintf("%s%s%d", c.mmUserID, subscriptionSuffix, time.Now().UnixNano()),
		Resource:           "caldav/poll",
		ChangeType:         "updated",
		NotificationURL:    notificationURL,
		ClientState:        ycalClientState,
		ExpirationDateTime: time.Now().Add(365 * 24 * time.Hour).Format(time.RFC3339),
		CreatorID:          remoteUserID,
	}
	return sub, nil
}

func (c *client) DeleteSubscription(sub *remote.Subscription) error {
	return nil
}

func (c *client) RenewSubscription(notificationURL, remoteUserID string, oldSub *remote.Subscription) (*remote.Subscription, error) {
	return c.CreateMySubscription(notificationURL, remoteUserID)
}

func (c *client) ListSubscriptions() ([]*remote.Subscription, error) {
	return nil, nil
}

func (c *client) GetNotificationData(orig *remote.Notification) (*remote.Notification, error) {
	return orig, nil
}

// PollNotifications compares current calendar state with the previous notification snapshot (stored events).
func (c *client) PollNotifications(remoteUserID, subscriptionID string) ([]*remote.Notification, error) {
	if subscriptionID == "" {
		return nil, nil
	}

	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now().Add(30 * 24 * time.Hour)

	events, err := c.queryRemoteEvents(start, end)
	if err != nil {
		return nil, errors.Wrap(err, "ycal PollNotifications")
	}

	var out []*remote.Notification
	for _, ev := range events {
		if ev == nil || ev.ICalUID == "" {
			continue
		}
		out = append(out, &remote.Notification{
			SubscriptionID: subscriptionID,
			Event:          ev,
			ClientState:    ycalClientState,
			ChangeType:     "updated",
			IsBare:         false,
		})
	}

	return out, nil
}
