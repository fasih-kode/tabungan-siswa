package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

func TestStudentLeavingFinalizeInputBeforeUOW(t *testing.T) {
	svc := &studentLeavingRequestService{}
	for _, role := range []domain.UserRole{domain.RoleWaliKelas, domain.RoleSiswa} {
		if _, err := svc.Finalize(context.Background(), FinalizeStudentLeavingRequestInput{Actor: Actor{UserID: uuid.New(), Role: role}, StudentLeavingRequestID: uuid.New()}); !errors.Is(err, ErrForbidden) {
			t.Fatalf("role=%s err=%v", role, err)
		}
	}
	if _, err := svc.Finalize(context.Background(), FinalizeStudentLeavingRequestInput{Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}}); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("id err=%v", err)
	}
	if _, err := svc.Finalize(context.Background(), FinalizeStudentLeavingRequestInput{Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, StudentLeavingRequestID: uuid.New()}); !errors.Is(err, ErrInvalidDependency) {
		t.Fatalf("dependency err=%v", err)
	}
}
