package secretcontract

import (
	"reflect"
	"testing"
)

func TestNames(t *testing.T) {
	got := Names()
	want := []string{
		TypeBasicCreds,
		TypeCertificate,
		TypeDatabaseCreds,
		TypeKeyValue,
		TypeOpaque,
		TypeSSHKey,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() mismatch: got=%v want=%v", got, want)
	}
}
