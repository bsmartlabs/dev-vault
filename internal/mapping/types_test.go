package mapping

import "testing"

func TestModeAllows(t *testing.T) {
	if !ModePull.AllowsPull() || ModePull.AllowsPush() {
		t.Fatal("ModePull should allow only pull")
	}
	if ModePush.AllowsPull() || !ModePush.AllowsPush() {
		t.Fatal("ModePush should allow only push")
	}
	if ModeSkip.AllowsPull() || ModeSkip.AllowsPush() || !ModeSkip.AllowsSkip() {
		t.Fatal("ModeSkip should allow only skip")
	}
	if !ModePull.IsSupportedCommandMode() || !ModePush.IsSupportedCommandMode() || ModeSkip.IsSupportedCommandMode() {
		t.Fatal("pull/push should be supported command modes")
	}
	if !ModePull.IsSupportedMappingMode() || !ModePush.IsSupportedMappingMode() || !ModeSkip.IsSupportedMappingMode() {
		t.Fatal("pull/push/skip should be supported mapping modes")
	}
	if Mode("").IsSupportedCommandMode() {
		t.Fatal("empty mode should not be supported")
	}
	if Mode("").IsSupportedMappingMode() {
		t.Fatal("empty mode should not be a supported mapping mode")
	}
	if !ModePull.AllowsCommand(ModePull) || ModePull.AllowsCommand(ModePush) {
		t.Fatal("ModePull command checks should allow only pull")
	}
	if !ModePush.AllowsCommand(ModePush) || ModePush.AllowsCommand(ModePull) {
		t.Fatal("ModePush command checks should allow only push")
	}
	if ModeSkip.AllowsCommand(ModePull) || ModeSkip.AllowsCommand(ModePush) {
		t.Fatal("ModeSkip command checks should not allow pull/push commands")
	}
}
