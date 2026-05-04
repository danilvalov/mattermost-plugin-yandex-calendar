package ycal

import (
	"context"
	"fmt"

	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
)

func (c *client) ensureCalendar(ctx context.Context) (*caldav.Client, string, error) {
	c.calMu.Lock()
	defer c.calMu.Unlock()

	if c.cachedCalDAV != nil && c.cachedCalPath != "" {
		return c.cachedCalDAV, c.cachedCalPath, nil
	}

	wdc, err := webdav.NewClient(c.httpClient, c.yandexEndpoint)
	if err != nil {
		return nil, "", err
	}
	principal, err := wdc.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("ycal: FindCurrentUserPrincipal: %w", err)
	}

	cd, err := caldav.NewClient(c.httpClient, c.yandexEndpoint)
	if err != nil {
		return nil, "", err
	}

	home, err := cd.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, "", fmt.Errorf("ycal: FindCalendarHomeSet: %w", err)
	}

	cals, err := cd.FindCalendars(ctx, home)
	if err != nil {
		return nil, "", fmt.Errorf("ycal: FindCalendars: %w", err)
	}
	if len(cals) == 0 {
		return nil, "", fmt.Errorf("ycal: no calendars found")
	}

	c.cachedCalDAV = cd
	c.cachedCalPath = cals[0].Path
	return c.cachedCalDAV, c.cachedCalPath, nil
}
