package presence

import "testing"

func TestStringer(t *testing.T) {
	e := &Event{}
	if e.Status.String() != unknown {
		t.Errorf("status string should be %s for uninitialized Event, got: %s ", unknown, e.Status.String())
	}

	e.Status = Online
	if e.Status.String() != online {
		t.Errorf("status string should be %s for online Event, got: %s ", online, e.Status.String())
	}

	e.Status = Offline
	if e.Status.String() != offline {
		t.Errorf("status string should be %s for offline Event, got: %s ", offline, e.Status.String())
	}
}
