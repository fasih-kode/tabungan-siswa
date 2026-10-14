package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestClassTransferServiceIntegration_RequestApproveAndReject(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()
	cleanupServiceScopeFixtures(t, db)

	repos := postgres.NewRepositorySet(db)
	uow := postgres.NewUnitOfWorkManager(db)
	admin := newServiceScopeUser(t, domain.RoleAdmin)
	if err := repos.Users.Create(ctx, admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	year := newServiceScopeAcademicYear(t, "Class transfer lifecycle "+uuid.NewString())
	if err := repos.AcademicYears.Create(ctx, year); err != nil {
		t.Fatalf("create academic year: %v", err)
	}
	fromClass := newServiceScopeClass(t, "Transfer source "+uuid.NewString(), 4)
	toClass := newServiceScopeClass(t, "Transfer destination "+uuid.NewString(), 5)
	if err := repos.Classes.Create(ctx, fromClass); err != nil {
		t.Fatalf("create source class: %v", err)
	}
	if err := repos.Classes.Create(ctx, toClass); err != nil {
		t.Fatalf("create destination class: %v", err)
	}
	students := []domain.Student{
		newServiceScopeStudent(t, "Transfer Approve Student"),
		newServiceScopeStudent(t, "Transfer Reject Student"),
	}
	for _, student := range students {
		if err := repos.Students.Create(ctx, student); err != nil {
			t.Fatalf("create student: %v", err)
		}
		if err := repos.StudentClassHistories.Create(ctx, newServiceScopeStudentHistory(t, student.ID, year.ID, fromClass.ID)); err != nil {
			t.Fatalf("create current class history: %v", err)
		}
	}

	svc, err := service.NewClassTransferService(service.Dependencies{Repositories: repos, UOW: uow})
	if err != nil {
		t.Fatalf("NewClassTransferService() error = %v", err)
	}
	actor := service.Actor{UserID: admin.ID, Role: domain.RoleAdmin}

	approvedRequest, err := svc.Request(ctx, service.RequestTransferInput{
		Actor: actor, StudentID: students[0].ID, AcademicYearID: year.ID,
		FromClassID: fromClass.ID, ToClassID: toClass.ID,
	})
	if err != nil {
		t.Fatalf("Request() for approval flow error = %v", err)
	}
	effectiveDate := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	approved, err := svc.Approve(ctx, service.ApproveTransferInput{
		Actor: actor, TransferRequestID: approvedRequest.TransferRequest.ID, EffectiveDate: effectiveDate,
	})
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if approved.TransferRequest.Status != domain.TransferApproved {
		t.Fatalf("approved request status = %q, want APPROVED", approved.TransferRequest.Status)
	}
	if approved.ClassHistory == nil || approved.ClassHistory.ClassID != toClass.ID || !approved.ClassHistory.Period.From.Equal(effectiveDate) {
		t.Fatalf("approved class history = %+v, want target class beginning on %s", approved.ClassHistory, effectiveDate.Format("2006-01-02"))
	}
	approvedHistory, err := repos.StudentClassHistories.ListByStudentAndAcademicYear(ctx, students[0].ID, year.ID, repository.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list approved student's class history: %v", err)
	}
	if len(approvedHistory) != 2 || approvedHistory[0].Period.To == nil || !approvedHistory[0].Period.To.Equal(effectiveDate) {
		t.Fatalf("approved student's class history = %+v, want closed source and new destination rows", approvedHistory)
	}

	rejectedRequest, err := svc.Request(ctx, service.RequestTransferInput{
		Actor: actor, StudentID: students[1].ID, AcademicYearID: year.ID,
		FromClassID: fromClass.ID, ToClassID: toClass.ID,
	})
	if err != nil {
		t.Fatalf("Request() for rejection flow error = %v", err)
	}
	reason := "Permintaan tidak sesuai jadwal kelas"
	rejected, err := svc.Reject(ctx, service.RejectTransferInput{
		Actor: actor, TransferRequestID: rejectedRequest.TransferRequest.ID, RejectionReason: reason,
	})
	if err != nil {
		t.Fatalf("Reject() error = %v", err)
	}
	if rejected.TransferRequest.Status != domain.TransferRejected || rejected.TransferRequest.RejectionReason == nil || *rejected.TransferRequest.RejectionReason != reason {
		t.Fatalf("rejected request = %+v, want REJECTED with reason", rejected.TransferRequest)
	}
	rejectedHistory, err := repos.StudentClassHistories.ListByStudentAndAcademicYear(ctx, students[1].ID, year.ID, repository.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list rejected student's class history: %v", err)
	}
	if len(rejectedHistory) != 1 || rejectedHistory[0].ClassID != fromClass.ID {
		t.Fatalf("rejected student's class history = %+v, want unchanged source history", rejectedHistory)
	}
}
