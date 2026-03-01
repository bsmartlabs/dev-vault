package mapping

import (
	"reflect"
	"strings"
	"testing"
)

func TestSelectTargetsForMode(t *testing.T) {
	entries := map[string]Entry{
		"a-dev": {Mode: ModePull, File: "a"},
		"b-dev": {Mode: ModePush, File: "b"},
		"c-dev": {Mode: ModePull, File: "c"},
	}

	t.Run("AllAndPositionalIsRejected", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, true, []string{"a-dev"}, ModePull); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("NoTargetsIsRejected", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, false, nil, ModePull); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("UnsupportedModeIsRejected", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, true, nil, Mode("nope")); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("AllModeFiltersAndSorts", func(t *testing.T) {
		got, err := SelectTargetsForMode(entries, true, nil, ModePull)
		if err != nil {
			t.Fatalf("SelectTargetsForMode: %v", err)
		}
		want := []Target{
			{Name: "a-dev", Entry: entries["a-dev"]},
			{Name: "c-dev", Entry: entries["c-dev"]},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected targets\nwant=%#v\ngot =%#v", want, got)
		}
	})

	t.Run("AllModeWithNoMatchesReturnsError", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, true, nil, Mode("pullx")); err == nil {
			t.Fatal("expected error")
		}
		_, err := SelectTargetsForMode(map[string]Entry{"a-dev": {Mode: ModePush}}, true, nil, ModePull)
		if err == nil || !strings.Contains(err.Error(), "no mapping entries selected") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ExplicitNamesDedupesPreservingOrder", func(t *testing.T) {
		got, err := SelectTargetsForMode(entries, false, []string{"c-dev", "a-dev", "c-dev"}, ModePull)
		if err != nil {
			t.Fatalf("SelectTargetsForMode: %v", err)
		}
		want := []Target{
			{Name: "c-dev", Entry: entries["c-dev"]},
			{Name: "a-dev", Entry: entries["a-dev"]},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected targets\nwant=%#v\ngot =%#v", want, got)
		}
	})

	t.Run("MissingSecretReturnsError", func(t *testing.T) {
		if _, err := SelectTargetsForMode(entries, false, []string{"missing-dev"}, ModePull); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ModeMismatchReturnsError", func(t *testing.T) {
		_, err := SelectTargetsForMode(entries, false, []string{"b-dev"}, ModePull)
		if err == nil || !strings.Contains(err.Error(), "cannot be used with pull") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
