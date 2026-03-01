package mapping

import "testing"

func TestModeAllows(t *testing.T) {
	if !ModePull.AllowsPull() || ModePull.AllowsPush() {
		t.Fatal("ModePull should allow only pull")
	}
	if ModePush.AllowsPull() || !ModePush.AllowsPush() {
		t.Fatal("ModePush should allow only push")
	}
	if !ModePull.IsSupportedCommandMode() || !ModePush.IsSupportedCommandMode() {
		t.Fatal("pull/push should be supported command modes")
	}
	if Mode("").IsSupportedCommandMode() {
		t.Fatal("empty mode should not be supported")
	}
	if !ModePull.AllowsCommand(ModePull) || ModePull.AllowsCommand(ModePush) {
		t.Fatal("ModePull command checks should allow only pull")
	}
	if !ModePush.AllowsCommand(ModePush) || ModePush.AllowsCommand(ModePull) {
		t.Fatal("ModePush command checks should allow only push")
	}
}
