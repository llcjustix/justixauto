package service

import (
	"strings"
	"testing"
)

type passwordErrors map[string]string

func (e passwordErrors) Add(field, message string) {
	e[field] = message
}

func TestValidatePassword(t *testing.T) {
	tooLong := strings.Repeat("a", 129)
	tests := []struct {
		name                     string
		field, password, confirm string
		want                     passwordErrors
	}{
		{name: "seven ASCII characters", field: "password", password: "1234567", confirm: "1234567", want: passwordErrors{"password": "at least 8 characters"}},
		{name: "short mismatching password keeps minimum error", field: "password", password: "1234567", confirm: "7654321", want: passwordErrors{"password": "at least 8 characters"}},
		{name: "eight ASCII characters", field: "password", password: "12345678", confirm: "12345678"},
		{name: "seven Unicode runes", field: "password", password: strings.Repeat("я", 7), confirm: strings.Repeat("я", 7), want: passwordErrors{"password": "at least 8 characters"}},
		{name: "eight Unicode runes", field: "password", password: strings.Repeat("я", 8), confirm: strings.Repeat("я", 8)},
		{name: "128 characters", field: "password", password: strings.Repeat("a", 128), confirm: strings.Repeat("a", 128)},
		{name: "129 characters", field: "password", password: tooLong, confirm: tooLong, want: passwordErrors{"password": "at most 128 characters"}},
		{name: "matching new password confirmation", field: "newPassword", password: "12345678", confirm: "12345678"},
		{name: "mismatching new password confirmation", field: "newPassword", password: "12345678", confirm: "87654321", want: passwordErrors{"newPasswordConfirmation": "does not match"}},
		{name: "first admin password field", field: "firstAdmin.password", password: "1234567", confirm: "1234567", want: passwordErrors{"firstAdmin.password": "at least 8 characters"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := passwordErrors{}
			validatePassword(got, tt.field, tt.password, tt.confirm)
			if len(got) != len(tt.want) {
				t.Fatalf("errors = %#v, want %#v", got, tt.want)
			}
			for field, want := range tt.want {
				if got[field] != want {
					t.Errorf("error for %q = %q, want %q", field, got[field], want)
				}
			}
		})
	}
}

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := hashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("unexpected hash format %q", hash)
	}
	if ok, err := verifyPassword("correct horse battery staple", hash); err != nil || !ok {
		t.Fatalf("correct password rejected: %v", err)
	}
	if ok, _ := verifyPassword("wrong password", hash); ok {
		t.Fatal("wrong password accepted")
	}
	other, _ := hashPassword("correct horse battery staple")
	if other == hash {
		t.Fatal("hashes must be salted")
	}
	for _, bad := range []string{"", "$argon2i$v=19$m=1,t=1,p=1$a$b", "$argon2id$v=19$m=x$a$b"} {
		if _, err := verifyPassword("x", bad); err == nil {
			t.Fatalf("malformed hash %q accepted", bad)
		}
	}
}
