package secrettype

import "testing"

func TestSecretTypeContract(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("expected canonical names")
	}
	for _, name := range names {
		if !IsValid(name) {
			t.Fatalf("expected %q to be valid", name)
		}
	}
	if IsValid("not-valid") {
		t.Fatal("unexpected valid type")
	}
}
