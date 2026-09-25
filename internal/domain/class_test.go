package domain

import "testing"

func TestNewClass(t *testing.T) {
	class, err := NewClass("VII A", 7)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	if class.ID == [16]byte{} {
		t.Fatal("class ID must not be nil")
	}

	if class.Name != "VII A" {
		t.Fatalf("Name = %q, want %q", class.Name, "VII A")
	}

	if class.Level != 7 {
		t.Fatalf("Level = %d, want 7", class.Level)
	}
}

func TestNewClassAllowsLevelOne(t *testing.T) {
	_, err := NewClass("I A", 1)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}
}

func TestNewClassAllowsLevelTwelve(t *testing.T) {
	_, err := NewClass("XII A", 12)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}
}

func TestNewClassRejectsLevelBelowOne(t *testing.T) {
	_, err := NewClass("VII A", 0)
	if err != ErrInvalidClassLevel {
		t.Fatalf("NewClass() error = %v, want %v", err, ErrInvalidClassLevel)
	}
}

func TestNewClassRejectsLevelAboveTwelve(t *testing.T) {
	_, err := NewClass("VII A", 13)
	if err != ErrInvalidClassLevel {
		t.Fatalf("NewClass() error = %v, want %v", err, ErrInvalidClassLevel)
	}
}

func TestNewClassRejectsEmptyName(t *testing.T) {
	_, err := NewClass("", 7)
	if err != ErrEmptyName {
		t.Fatalf("NewClass() error = %v, want %v", err, ErrEmptyName)
	}
}

func TestNewClassRejectsWhitespaceName(t *testing.T) {
	_, err := NewClass("   ", 7)
	if err != ErrEmptyName {
		t.Fatalf("NewClass() error = %v, want %v", err, ErrEmptyName)
	}
}
