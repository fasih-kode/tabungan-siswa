package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewStudentClassHistory(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()
	classID := uuid.New()

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)

	period, err := NewDateRange(from, &to)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	history, err := NewStudentClassHistory(
		studentID,
		academicYearID,
		classID,
		period,
	)
	if err != nil {
		t.Fatalf("NewStudentClassHistory() error = %v", err)
	}

	if history.ID == uuid.Nil {
		t.Fatal("history ID must not be nil")
	}

	if history.StudentID != studentID {
		t.Fatalf("StudentID = %v, want %v", history.StudentID, studentID)
	}

	if history.AcademicYearID != academicYearID {
		t.Fatalf(
			"AcademicYearID = %v, want %v",
			history.AcademicYearID,
			academicYearID,
		)
	}

	if history.ClassID != classID {
		t.Fatalf("ClassID = %v, want %v", history.ClassID, classID)
	}

	if !history.Period.From.Equal(from) {
		t.Fatalf(
			"Period.From = %v, want %v",
			history.Period.From,
			from,
		)
	}

	if history.Period.To == nil {
		t.Fatal("Period.To must not be nil")
	}

	if !history.Period.To.Equal(to) {
		t.Fatalf(
			"Period.To = %v, want %v",
			*history.Period.To,
			to,
		)
	}
}

func TestNewStudentClassHistoryRejectsNilStudentID(t *testing.T) {
	period, err := NewDateRange(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	_, err = NewStudentClassHistory(
		uuid.Nil,
		uuid.New(),
		uuid.New(),
		period,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewStudentClassHistory() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewStudentClassHistoryRejectsNilAcademicYearID(t *testing.T) {
	period, err := NewDateRange(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	_, err = NewStudentClassHistory(
		uuid.New(),
		uuid.Nil,
		uuid.New(),
		period,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewStudentClassHistory() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewStudentClassHistoryRejectsNilClassID(t *testing.T) {
	period, err := NewDateRange(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	_, err = NewStudentClassHistory(
		uuid.New(),
		uuid.New(),
		uuid.Nil,
		period,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewStudentClassHistory() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestStudentClassHistoryValidateWithinAcademicYear(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	insideEnd := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	outsideStart := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	valid, err := NewStudentClassHistory(uuid.New(), uuid.New(), uuid.New(), DateRange{From: start, To: &end})
	if err != nil {
		t.Fatalf("NewStudentClassHistory(valid) error = %v", err)
	}
	if err := valid.ValidateWithinAcademicYear(start, end); err != nil {
		t.Fatalf("ValidateWithinAcademicYear(valid) error = %v", err)
	}

	before, err := NewStudentClassHistory(uuid.New(), uuid.New(), uuid.New(), DateRange{From: outsideStart, To: &insideEnd})
	if err != nil {
		t.Fatalf("NewStudentClassHistory(before) error = %v", err)
	}
	if err := before.ValidateWithinAcademicYear(start, end); err != ErrInvalidDateRange {
		t.Fatalf("ValidateWithinAcademicYear(before) error = %v, want %v", err, ErrInvalidDateRange)
	}

	openEnded, err := NewStudentClassHistory(uuid.New(), uuid.New(), uuid.New(), DateRange{From: start})
	if err != nil {
		t.Fatalf("NewStudentClassHistory(open-ended) error = %v", err)
	}
	if err := openEnded.ValidateWithinAcademicYear(start, end); err != ErrInvalidDateRange {
		t.Fatalf("ValidateWithinAcademicYear(open-ended) error = %v, want %v", err, ErrInvalidDateRange)
	}
}

func TestStudentClassHistoryContains(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)

	period, err := NewDateRange(from, &to)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	history, err := NewStudentClassHistory(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		period,
	)
	if err != nil {
		t.Fatalf("NewStudentClassHistory() error = %v", err)
	}

	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{
			name: "before period",
			date: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "at start",
			date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "inside period",
			date: time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "at end",
			date: time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := history.Contains(tt.date); got != tt.want {
				t.Fatalf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewStudentClassHistoryAllowsOpenEndedPeriod(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	period, err := NewDateRange(from, nil)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	history, err := NewStudentClassHistory(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		period,
	)
	if err != nil {
		t.Fatalf("NewStudentClassHistory() error = %v", err)
	}

	if history.Period.To != nil {
		t.Fatal("Period.To must be nil for an open-ended period")
	}

	date := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	if !history.Contains(date) {
		t.Fatal("open-ended history should contain dates after its start")
	}
}
