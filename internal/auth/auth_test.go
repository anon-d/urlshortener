package auth

import (
	"testing"
)

func TestGenerateUserID_Unique(t *testing.T) {
	id1 := GenerateUserID()
	id2 := GenerateUserID()

	if id1 == "" {
		t.Error("expected non-empty user ID")
	}
	if id1 == id2 {
		t.Error("expected different user IDs, got identical")
	}
}

func TestGenerateUserID_NotEmpty(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := GenerateUserID()
		if id == "" {
			t.Fatalf("iteration %d: got empty user ID", i)
		}
	}
}

func TestSignValue_Deterministic(t *testing.T) {
	value := "test-user-id"
	secret := "my-secret"

	signed1 := SignValue(value, secret)
	signed2 := SignValue(value, secret)

	if signed1 != signed2 {
		t.Errorf("expected same signature for same input, got %q and %q", signed1, signed2)
	}
}

func TestSignValue_ContainsOriginal(t *testing.T) {
	value := "test-user-id"
	signed := SignValue(value, "secret")

	if len(signed) <= len(value) {
		t.Errorf("signed value %q should be longer than original %q", signed, value)
	}
}

func TestSignValue_DifferentSecrets(t *testing.T) {
	value := "test-user-id"
	signed1 := SignValue(value, "secret1")
	signed2 := SignValue(value, "secret2")

	if signed1 == signed2 {
		t.Error("expected different signatures for different secrets")
	}
}

func TestValidateSignedValue_Valid(t *testing.T) {
	value := "test-user-id"
	secret := "my-secret"
	signed := SignValue(value, secret)

	got, ok := ValidateSignedValue(signed, secret)
	if !ok {
		t.Error("expected valid signature")
	}
	if got != value {
		t.Errorf("expected %q, got %q", value, got)
	}
}

func TestValidateSignedValue_WrongSecret(t *testing.T) {
	signed := SignValue("test-user-id", "secret1")

	_, ok := ValidateSignedValue(signed, "secret2")
	if ok {
		t.Error("expected invalid signature with wrong secret")
	}
}

func TestValidateSignedValue_Tampered(t *testing.T) {
	signed := SignValue("test-user-id", "secret")
	tampered := signed + "x"

	_, ok := ValidateSignedValue(tampered, "secret")
	if ok {
		t.Error("expected invalid signature for tampered value")
	}
}

func TestValidateSignedValue_NoDot(t *testing.T) {
	_, ok := ValidateSignedValue("no-dot-here", "secret")
	if ok {
		t.Error("expected invalid for value without dot separator")
	}
}

func TestValidateSignedValue_Empty(t *testing.T) {
	_, ok := ValidateSignedValue("", "secret")
	if ok {
		t.Error("expected invalid for empty value")
	}
}
