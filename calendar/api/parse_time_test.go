package api

import (
	"testing"
	"time"
)

func TestParseRFC3339OrDate(t *testing.T) {
	t.Parallel()

	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		in      string
		loc     *time.Location
		wantAll bool
		wantH   int
		wantM   int
	}{
		{name: "date only", in: "2026-07-28", loc: msk, wantAll: true},
		{name: "rfc3339 z", in: "2026-07-28T19:00:00Z", loc: msk, wantH: 22, wantM: 0},
		{name: "wall clock mui", in: "2026-07-28T22:00:00", loc: msk, wantH: 22, wantM: 0},
		{name: "wall clock no seconds", in: "2026-07-28T22:30", loc: msk, wantH: 22, wantM: 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, allDay, err := parseRFC3339OrDate(tc.in, tc.loc)
			if err != nil {
				t.Fatal(err)
			}
			if allDay != tc.wantAll {
				t.Fatalf("allDay=%v want %v", allDay, tc.wantAll)
			}
			if tc.wantAll {
				return
			}
			local := got.In(tc.loc)
			if local.Hour() != tc.wantH || local.Minute() != tc.wantM {
				t.Fatalf("got %s want %02d:%02d in %s", local.Format(time.RFC3339), tc.wantH, tc.wantM, tc.loc)
			}
		})
	}

	t.Run("invalid", func(t *testing.T) {
		if _, _, err := parseRFC3339OrDate("not-a-time", msk); err == nil {
			t.Fatal("expected error")
		}
	})
}
