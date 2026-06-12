package domain

import (
	"errors"
	"testing"
)

func TestStudent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		student Student
		wantErr bool
	}{
		{"valid", Student{Name: "Іван", Email: "ivan@lnu.ua"}, false},
		{"valid minimal email", Student{Name: "A", Email: "a@b"}, false},
		{"valid name with surrounding spaces", Student{Name: "  Bob  ", Email: "bob@x.io"}, false},
		{"valid unicode name", Student{Name: "Олександра", Email: "olya@lnu.ua"}, false},
		{"empty name", Student{Name: "", Email: "x@y.z"}, true},
		{"whitespace-only name", Student{Name: "   ", Email: "x@y.z"}, true},
		{"tab/newline name", Student{Name: "\t\n", Email: "x@y.z"}, true},
		{"email without at sign", Student{Name: "Bob", Email: "bob.example.com"}, true},
		{"empty email", Student{Name: "Bob", Email: ""}, true},
		{"both invalid reports name first", Student{Name: "", Email: "nope"}, true},
		{"at sign only is accepted", Student{Name: "Bob", Email: "@"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.student.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Validate() = nil, want error")
				}
				if !errors.Is(err, ErrValidation) {
					t.Errorf("Validate() error = %v, want errors.Is ErrValidation", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestValidationError_Is(t *testing.T) {
	err := (&Student{Name: "", Email: "x"}).Validate()

	tests := []struct {
		name   string
		target error
		want   bool
	}{
		{"matches ErrValidation", ErrValidation, true},
		{"does not match ErrNotFound", ErrNotFound, false},
		{"does not match ErrDuplicate", ErrDuplicate, false},
		{"does not match ErrAlreadyEnrolled", ErrAlreadyEnrolled, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(err, tt.target); got != tt.want {
				t.Errorf("errors.Is(err, %v) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}

func TestValidationError_Message(t *testing.T) {
	err := (&Student{Name: "Bob", Email: "no-at-sign"}).Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	const want = "validation failed: valid email is required"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := map[string]error{
		"ErrNotFound":        ErrNotFound,
		"ErrDuplicate":       ErrDuplicate,
		"ErrValidation":      ErrValidation,
		"ErrAlreadyEnrolled": ErrAlreadyEnrolled,
	}
	for name, err := range sentinels {
		t.Run(name, func(t *testing.T) {
			if err == nil {
				t.Fatalf("%s is nil", name)
			}
			if err.Error() == "" {
				t.Errorf("%s has empty message", name)
			}
			for otherName, other := range sentinels {
				if otherName == name {
					continue
				}
				if errors.Is(err, other) {
					t.Errorf("%s should not match %s", name, otherName)
				}
			}
		})
	}
}
