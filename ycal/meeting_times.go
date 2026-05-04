package ycal

import (
	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func (c *client) FindMeetingTimes(params *remote.FindMeetingTimesParameters) (*remote.MeetingTimeSuggestionResults, error) {
	return nil, errors.New("ycal FindMeetingTimes not implemented")
}
