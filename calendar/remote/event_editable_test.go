package remote

import "testing"

func TestEventEditable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ev   *Event
		want bool
	}{
		{name: "nil", ev: nil, want: false},
		{name: "attendee", ev: &Event{IsOrganizer: false}, want: false},
		{name: "organizer", ev: &Event{IsOrganizer: true}, want: true},
		{name: "cancelled", ev: &Event{IsOrganizer: true, IsCancelled: true}, want: false},
		{name: "recurring", ev: &Event{IsOrganizer: true, IsRecurring: true}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.ev.Editable(); got != tc.want {
				t.Fatalf("Editable()=%v want %v", got, tc.want)
			}
		})
	}
}
