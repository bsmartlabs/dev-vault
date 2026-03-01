package secretcontract

import (
	"reflect"
	"testing"
)

func TestNames(t *testing.T) {
	got := Names()
	want := []string{
		string(TypeBasicCreds),
		string(TypeCertificate),
		string(TypeDatabaseCreds),
		string(TypeKeyValue),
		string(TypeOpaque),
		string(TypeSSHKey),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() mismatch: got=%v want=%v", got, want)
	}
}

func TestDevNamePolicy(t *testing.T) {
	if !IsDevSecretName("x-dev") {
		t.Fatal("expected x-dev to be recognized as dev secret")
	}
	if IsDevSecretName("x-prod") {
		t.Fatal("expected x-prod to be rejected as dev secret")
	}
	if err := ValidateDevSecretName("x-dev"); err != nil {
		t.Fatalf("expected validation success, got %v", err)
	}
	if err := ValidateDevSecretName("x-prod"); err == nil {
		t.Fatal("expected validation error for non-dev name")
	}
}
