package service

import (
	"context"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

// StudentLeavingRequestService menyediakan lifecycle proses pindah untuk pemanggil internal.
// Service belum dipasang pada ServiceSet atau endpoint aplikasi.
type StudentLeavingRequestService interface {
	// Start menerima tanggal hari ini atau tanggal lampau dalam tahun OPEN jika placement berlaku.
	// Tanggal kalender memakai WIB; timestamp audit tetap UTC.
	Start(context.Context, StartStudentLeavingRequestInput) (StartStudentLeavingRequestOutput, error)
	Get(context.Context, GetStudentLeavingRequestInput) (GetStudentLeavingRequestOutput, error)
	Cancel(context.Context, CancelStudentLeavingRequestInput) (CancelStudentLeavingRequestOutput, error)
}

type StartStudentLeavingRequestInput struct {
	Actor                Actor
	StudentID            uuid.UUID
	AcademicYearID       uuid.UUID
	EffectiveLeavingDate time.Time
}

type StartStudentLeavingRequestOutput struct {
	Request *domain.StudentLeavingRequest
}

type GetStudentLeavingRequestInput struct {
	Actor     Actor
	RequestID uuid.UUID
}

type GetStudentLeavingRequestOutput struct {
	Request *domain.StudentLeavingRequest
}

type CancelStudentLeavingRequestInput struct {
	Actor                   Actor
	StudentLeavingRequestID uuid.UUID
	Reason                  string
}

type CancelStudentLeavingRequestOutput struct {
	Request *domain.StudentLeavingRequest
}
