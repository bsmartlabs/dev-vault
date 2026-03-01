package mapping

import "testing"

func TestModeAllows(t *testing.T) {
	if !ModePull.AllowsPull() || ModePull.AllowsPush() {
		t.Fatal("ModePull should allow only pull")
	}
	if ModePush.AllowsPull() || !ModePush.AllowsPush() {
		t.Fatal("ModePush should allow only push")
	}
}
