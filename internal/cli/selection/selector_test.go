package selection

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
)

func TestSelectTargetsForMode(t *testing.T) {
	entries := map[string]mapping.Entry{
		"a-dev": {Mode: mapping.ModePull, File: "a"},
		"b-dev": {Mode: mapping.ModePush, File: "b"},
		"c-dev": {Mode: mapping.ModePull, File: "c"},
		"d-dev": {Mode: mapping.ModeSkip, File: "d"},
	}

	t.Run("AllAndPositionalIsRejected", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, true, []string{"a-dev"}, mapping.ModePull); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("NoTargetsIsRejected", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, false, nil, mapping.ModePull); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("UnsupportedModeIsRejected", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, true, nil, mapping.Mode("nope")); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("AllModeFiltersAndSorts", func(t *testing.T) {
		got, err := SelectTargetsForMode(entries, true, nil, mapping.ModePull)
		if err != nil {
			t.Fatalf("SelectTargetsForMode: %v", err)
		}
		want := []mapping.Target{
			{Name: "a-dev", Entry: entries["a-dev"]},
			{Name: "c-dev", Entry: entries["c-dev"]},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected targets\nwant=%#v\ngot =%#v", want, got)
		}
	})

	t.Run("AllModeWithNoMatchesReturnsError", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, true, nil, mapping.Mode("pullx")); err == nil {
			t.Fatal("expected error")
		}
		_, err := SelectTargetsForMode(map[string]mapping.Entry{"a-dev": {Mode: mapping.ModePush}}, true, nil, mapping.ModePull)
		if err == nil || !strings.Contains(err.Error(), "no mapping entries selected") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ExplicitNamesDedupesPreservingOrder", func(t *testing.T) {
		got, err := SelectTargetsForMode(entries, false, []string{"c-dev", "a-dev", "c-dev"}, mapping.ModePull)
		if err != nil {
			t.Fatalf("SelectTargetsForMode: %v", err)
		}
		want := []mapping.Target{
			{Name: "c-dev", Entry: entries["c-dev"]},
			{Name: "a-dev", Entry: entries["a-dev"]},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected targets\nwant=%#v\ngot =%#v", want, got)
		}
	})

	t.Run("MissingSecretReturnsError", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, false, []string{"missing-dev"}, mapping.ModePull); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ModeMismatchReturnsError", func(t *testing.T) {
		_, err := SelectTargetsForMode(entries, false, []string{"b-dev"}, mapping.ModePull)
		if err == nil || !strings.Contains(err.Error(), "cannot be used with pull") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("SkipModeIsExcludedFromAll", func(t *testing.T) {
		gotPull, err := SelectTargetsForMode(entries, true, nil, mapping.ModePull)
		if err != nil {
			t.Fatalf("SelectTargetsForMode pull --all: %v", err)
		}
		for _, target := range gotPull {
			if target.Name == "d-dev" {
				t.Fatal("skip mode entry should not be selected by pull --all")
			}
		}

		gotPush, err := SelectTargetsForMode(entries, true, nil, mapping.ModePush)
		if err != nil {
			t.Fatalf("SelectTargetsForMode push --all: %v", err)
		}
		for _, target := range gotPush {
			if target.Name == "d-dev" {
				t.Fatal("skip mode entry should not be selected by push --all")
			}
		}
	})

	t.Run("ExplicitSkipModeReturnsModeMismatch", func(t *testing.T) {
		_, err := SelectTargetsForMode(entries, false, []string{"d-dev"}, mapping.ModePull)
		if err == nil || !strings.Contains(err.Error(), "mode=skip") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
