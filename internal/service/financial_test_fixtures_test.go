package service

import (
	"context"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var financialTestTeacherID = uuid.MustParse("a39eaf72-a984-4c40-a8b4-fefbcd53ab36")

// Financial operation tests use one assigned teacher. Scope-specific tests keep
// their explicit repositories so missing or unrelated assignments remain denied.
func financialTestScope(repos repository.RepositorySet) repository.RepositorySet {
	classID := uuid.MustParse("a39eaf72-a984-4c40-a8b4-fefbcd53ab37")
	repos.StudentClassHistories = &savingsAccountServiceFakeStudentClassHistoryRepository{listFn: func(_ context.Context, studentID, yearID uuid.UUID, _ repository.ListOptions) ([]domain.StudentClassHistory, error) {
		return []domain.StudentClassHistory{{StudentID: studentID, AcademicYearID: yearID, ClassID: classID}}, nil
	}}
	repos.TeacherClassAssignments = &savingsAccountServiceFakeTeacherAssignmentRepository{listByClassFn: func(_ context.Context, requestedClassID, yearID uuid.UUID, _ repository.ListOptions) ([]domain.TeacherClassAssignment, error) {
		if requestedClassID != classID {
			return nil, nil
		}
		return []domain.TeacherClassAssignment{{UserID: financialTestTeacherID, AcademicYearID: yearID, ClassID: classID}}, nil
	}}
	return repos
}

func newScopedSavingsAccountServiceForTest(t *testing.T, repos repository.RepositorySet, manager repository.UnitOfWorkManager) *savingsAccountService {
	t.Helper()
	if m, ok := manager.(*savingsAccountServiceFakeUnitOfWorkManager); ok {
		if u, ok := m.uow.(*savingsAccountServiceFakeUnitOfWork); ok {
			u.repos = financialTestScope(u.repos)
		}
	}
	return newSavingsAccountServiceForTest(t, financialTestScope(repos), manager)
}

func newScopedTransactionServiceForTest(t *testing.T, repos repository.RepositorySet, manager repository.UnitOfWorkManager) *transactionService {
	t.Helper()
	if m, ok := manager.(*transactionServiceFakeUnitOfWorkManager); ok {
		if u, ok := m.uow.(*transactionServiceFakeUnitOfWork); ok {
			u.repos = financialTestScope(u.repos)
		}
	}
	return newTransactionServiceForTest(t, financialTestScope(repos), manager)
}

func newScopedSettlementServiceForTest(t *testing.T, repos repository.RepositorySet, manager repository.UnitOfWorkManager) *settlementService {
	t.Helper()
	if m, ok := manager.(*settlementServiceFakeUnitOfWorkManager); ok {
		if u, ok := m.uow.(*settlementServiceFakeUnitOfWork); ok {
			u.repos = financialTestScope(u.repos)
		}
	}
	return newSettlementServiceForTest(t, financialTestScope(repos), manager)
}
