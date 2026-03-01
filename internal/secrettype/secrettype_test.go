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

	for _, name := range names {
		if _, err := ToScaleway(name); err != nil {
			t.Fatalf("expected scaleway mapping for %q: %v", name, err)
		}
	}
	if _, err := ToScaleway("not-valid"); err == nil {
		t.Fatal("expected mapping error for unsupported type")
	}
}
