package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewStudent(t *testing.T) {
	userID := uuid.New()
	nis := "12345"
	nisn := "0012345678"

	student, err := NewStudent(
		"Ahmad Fauzan",
		&userID,
		&nis,
		&nisn,
	)
	if err != nil {
		t.Fatalf("NewStudent() error = %v", err)
	}

	if student.ID == uuid.Nil {
		t.Fatal("student ID must not be nil")
	}

	if student.UserID == nil || *student.UserID != userID {
		t.Fatalf("UserID = %v, want %v", student.UserID, userID)
	}

	if student.NIS == nil || *student.NIS != nis {
		t.Fatalf("NIS = %v, want %v", student.NIS, nis)
	}

	if student.NISN == nil || *student.NISN != nisn {
		t.Fatalf("NISN = %v, want %v", student.NISN, nisn)
	}

	if student.Name != "Ahmad Fauzan" {
		t.Fatalf("Name = %q, want %q", student.Name, "Ahmad Fauzan")
	}

	if student.Status != StudentActive {
		t.Fatalf("Status = %v, want %v", student.Status, StudentActive)
	}

	if !student.IsActive() {
		t.Fatal("student should be active")
	}

	if student.IsLeft() {
		t.Fatal("student should not be left")
	}
}

func TestNewStudentAllowsOptionalIdentityFields(t *testing.T) {
	student, err := NewStudent(
		"Ahmad Fauzan",
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewStudent() error = %v", err)
	}

	if student.UserID != nil {
		t.Fatal("UserID should be nil")
	}

	if student.NIS != nil {
		t.Fatal("NIS should be nil")
	}

	if student.NISN != nil {
		t.Fatal("NISN should be nil")
	}
}

func TestNewStudentRejectsEmptyName(t *testing.T) {
	_, err := NewStudent("", nil, nil, nil)
	if err != ErrEmptyName {
		t.Fatalf("NewStudent() error = %v, want %v", err, ErrEmptyName)
	}
}

func TestNewStudentRejectsWhitespaceName(t *testing.T) {
	_, err := NewStudent("   ", nil, nil, nil)
	if err != ErrEmptyName {
		t.Fatalf("NewStudent() error = %v, want %v", err, ErrEmptyName)
	}
}

func TestStudentLeave(t *testing.T) {
	student, err := NewStudent(
		"Ahmad Fauzan",
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewStudent() error = %v", err)
	}

	if err := student.Leave(); err != nil {
		t.Fatalf("Leave() error = %v", err)
	}

	if student.Status != StudentLeft {
		t.Fatalf("Status = %v, want %v", student.Status, StudentLeft)
	}

	if student.IsActive() {
		t.Fatal("student should not be active")
	}

	if !student.IsLeft() {
		t.Fatal("student should be left")
	}
}

func TestStudentCannotLeaveTwice(t *testing.T) {
	student, err := NewStudent(
		"Ahmad Fauzan",
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewStudent() error = %v", err)
	}

	if err := student.Leave(); err != nil {
		t.Fatalf("first Leave() error = %v", err)
	}

	if err := student.Leave(); err != ErrInvalidTransition {
		t.Fatalf("second Leave() error = %v, want %v", err, ErrInvalidTransition)
	}
}
