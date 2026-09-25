package domain

import (
	"testing"
	"time"
)

func TestNewAcademicYear(t *testing.T) {
	startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)

	academicYear, err := NewAcademicYear(
		"2026/2027",
		startDate,
		endDate,
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	if academicYear.ID == [16]byte{} {
		t.Fatal("academic year ID must not be nil")
	}

	if academicYear.Name != "2026/2027" {
		t.Fatalf("Name = %q, want %q", academicYear.Name, "2026/2027")
	}

	if !academicYear.StartDate.Equal(startDate) {
		t.Fatalf("StartDate = %v, want %v", academicYear.StartDate, startDate)
	}

	if !academicYear.EndDate.Equal(endDate) {
		t.Fatalf("EndDate = %v, want %v", academicYear.EndDate, endDate)
	}

	if academicYear.Status != AcademicYearOpen {
		t.Fatalf("Status = %v, want %v", academicYear.Status, AcademicYearOpen)
	}

	if !academicYear.IsOpen() {
		t.Fatal("academic year should be open")
	}
}

func TestNewAcademicYearTrimsName(t *testing.T) {
	startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)

	academicYear, err := NewAcademicYear(
		"  2026/2027  ",
		startDate,
		endDate,
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	if academicYear.Name != "2026/2027" {
		t.Fatalf(
			"Name = %q, want %q",
			academicYear.Name,
			"2026/2027",
		)
	}
}

func TestNewAcademicYearRejectsEmptyName(t *testing.T) {
	startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)

	_, err := NewAcademicYear("", startDate, endDate)
	if err != ErrEmptyName {
		t.Fatalf("NewAcademicYear() error = %v, want %v", err, ErrEmptyName)
	}
}

func TestNewAcademicYearRejectsWhitespaceOnlyName(t *testing.T) {
	startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)

	_, err := NewAcademicYear("   ", startDate, endDate)
	if err != ErrEmptyName {
		t.Fatalf(
			"NewAcademicYear() error = %v, want %v",
			err,
			ErrEmptyName,
		)
	}
}

func TestNewAcademicYearRejectsInvalidDateRange(t *testing.T) {
	startDate := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	_, err := NewAcademicYear("2026/2027", startDate, endDate)
	if err != ErrInvalidDateRange {
		t.Fatalf("NewAcademicYear() error = %v, want %v", err, ErrInvalidDateRange)
	}
}

func TestNewAcademicYearRejectsSameDate(t *testing.T) {
	date := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	_, err := NewAcademicYear("2026/2027", date, date)
	if err != ErrInvalidDateRange {
		t.Fatalf("NewAcademicYear() error = %v, want %v", err, ErrInvalidDateRange)
	}
}

func TestAcademicYearStartClosing(t *testing.T) {
	academicYear, err := NewAcademicYear(
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	if err := academicYear.StartClosing(); err != nil {
		t.Fatalf("StartClosing() error = %v", err)
	}

	if !academicYear.IsClosing() {
		t.Fatal("academic year should be closing")
	}
}

func TestAcademicYearCannotStartClosingTwice(t *testing.T) {
	academicYear, err := NewAcademicYear(
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	if err := academicYear.StartClosing(); err != nil {
		t.Fatalf("first StartClosing() error = %v", err)
	}

	if err := academicYear.StartClosing(); err != ErrInvalidTransition {
		t.Fatalf("second StartClosing() error = %v, want %v", err, ErrInvalidTransition)
	}
}

func TestAcademicYearClose(t *testing.T) {
	academicYear, err := NewAcademicYear(
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	if err := academicYear.StartClosing(); err != nil {
		t.Fatalf("StartClosing() error = %v", err)
	}

	if err := academicYear.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !academicYear.IsClosed() {
		t.Fatal("academic year should be closed")
	}
}

func TestAcademicYearCannotCloseWhileOpen(t *testing.T) {
	academicYear, err := NewAcademicYear(
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	if err := academicYear.Close(); err != ErrInvalidTransition {
		t.Fatalf("Close() error = %v, want %v", err, ErrInvalidTransition)
	}
}

func TestAcademicYearReopen(t *testing.T) {
	academicYear, err := NewAcademicYear(
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	if err := academicYear.StartClosing(); err != nil {
		t.Fatalf("StartClosing() error = %v", err)
	}

	if err := academicYear.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if err := academicYear.Reopen(); err != nil {
		t.Fatalf("Reopen() error = %v", err)
	}

	if !academicYear.IsOpen() {
		t.Fatal("academic year should be open after reopen")
	}
}

func TestAcademicYearCannotReopenWhileOpen(t *testing.T) {
	academicYear, err := NewAcademicYear(
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	if err := academicYear.Reopen(); err != ErrInvalidTransition {
		t.Fatalf("Reopen() error = %v, want %v", err, ErrInvalidTransition)
	}
}

func TestAcademicYearCannotCloseFromOpenDirectly(t *testing.T) {
	academicYear, err := NewAcademicYear(
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	if academicYear.Status != AcademicYearOpen {
		t.Fatalf("Status = %v, want %v", academicYear.Status, AcademicYearOpen)
	}

	if err := academicYear.Close(); err != ErrInvalidTransition {
		t.Fatalf("Close() error = %v, want %v", err, ErrInvalidTransition)
	}
}
