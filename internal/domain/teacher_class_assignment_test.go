package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewTeacherClassAssignment(t *testing.T) {
	userID := uuid.New()
	academicYearID := uuid.New()
	classID := uuid.New()

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)

	period, err := NewDateRange(from, &to)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	assignment, err := NewTeacherClassAssignment(
		userID,
		academicYearID,
		classID,
		period,
	)
	if err != nil {
		t.Fatalf("NewTeacherClassAssignment() error = %v", err)
	}

	if assignment.ID == uuid.Nil {
		t.Fatal("assignment ID must not be nil")
	}

	if assignment.UserID != userID {
		t.Fatalf("UserID = %v, want %v", assignment.UserID, userID)
	}

	if assignment.AcademicYearID != academicYearID {
		t.Fatalf(
			"AcademicYearID = %v, want %v",
			assignment.AcademicYearID,
			academicYearID,
		)
	}

	if assignment.ClassID != classID {
		t.Fatalf("ClassID = %v, want %v", assignment.ClassID, classID)
	}

	if !assignment.Period.From.Equal(from) {
		t.Fatalf(
			"Period.From = %v, want %v",
			assignment.Period.From,
			from,
		)
	}

	if assignment.Period.To == nil {
		t.Fatal("Period.To must not be nil")
	}

	if !assignment.Period.To.Equal(to) {
		t.Fatalf(
			"Period.To = %v, want %v",
			*assignment.Period.To,
			to,
		)
	}
}

func TestNewTeacherClassAssignmentRejectsNilUserID(t *testing.T) {
	period, err := NewDateRange(
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	_, err = NewTeacherClassAssignment(
		uuid.Nil,
		uuid.New(),
		uuid.New(),
		period,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewTeacherClassAssignment() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewTeacherClassAssignmentRejectsNilAcademicYearID(t *testing.T) {
	period, err := NewDateRange(
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	_, err = NewTeacherClassAssignment(
		uuid.New(),
		uuid.Nil,
		uuid.New(),
		period,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewTeacherClassAssignment() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewTeacherClassAssignmentRejectsNilClassID(t *testing.T) {
	period, err := NewDateRange(
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	_, err = NewTeacherClassAssignment(
		uuid.New(),
		uuid.New(),
		uuid.Nil,
		period,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewTeacherClassAssignment() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestTeacherClassAssignmentContains(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)

	period, err := NewDateRange(from, &to)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	assignment, err := NewTeacherClassAssignment(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		period,
	)
	if err != nil {
		t.Fatalf("NewTeacherClassAssignment() error = %v", err)
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
			if got := assignment.Contains(tt.date); got != tt.want {
				t.Fatalf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTeacherClassAssignmentAllowsOpenEndedPeriod(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	period, err := NewDateRange(from, nil)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	assignment, err := NewTeacherClassAssignment(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		period,
	)
	if err != nil {
		t.Fatalf("NewTeacherClassAssignment() error = %v", err)
	}

	if assignment.Period.To != nil {
		t.Fatal("Period.To must be nil for an open-ended period")
	}

	date := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	if !assignment.Contains(date) {
		t.Fatal("open-ended assignment should contain dates after its start")
	}
}
