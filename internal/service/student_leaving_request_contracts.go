package service

import (
	"context"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

// StudentLeavingRequestService menyediakan awal proses pindah untuk pemanggil internal.
// Service belum dipasang pada ServiceSet atau endpoint aplikasi.
type StudentLeavingRequestService interface {
	// Start sementara dibatasi pada tahun OPEN dan tanggal hari ini UTC dengan placement yang berlaku.
	// Batas tanggal ini perlu review kebijakan sebelum integrasi ke jalur pengguna.
	Start(context.Context, StartStudentLeavingRequestInput) (StartStudentLeavingRequestOutput, error)
	Get(context.Context, GetStudentLeavingRequestInput) (GetStudentLeavingRequestOutput, error)
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
