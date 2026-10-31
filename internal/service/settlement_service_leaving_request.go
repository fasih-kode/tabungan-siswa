package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type settleForStudentLeavingRequestInput struct {
	Actor                   Actor
	StudentLeavingRequestID uuid.UUID
	SavingsAccountID        uuid.UUID
	ExpectedBalance         domain.Money
}

// settleForStudentLeavingRequest menyelesaikan satu rekening pada proses pindah aktif.
// Method ini sengaja tidak diekspos melalui SettlementService atau ServiceSet sebelum alur HTTP ditinjau.
func (s *settlementService) settleForStudentLeavingRequest(
	ctx context.Context,
	input settleForStudentLeavingRequestInput,
) (SettleStudentLeavingOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleWaliKelas); err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if input.StudentLeavingRequestID == uuid.Nil || input.SavingsAccountID == uuid.Nil {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidID
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return SettleStudentLeavingOutput{}, fmt.Errorf("begin leaving request settlement: %w", err)
	}
	ctx = txCtx
	committed := false
	defer func() {
		if ownsTransaction && !committed {
			_ = uow.Rollback()
		}
	}()

	initialAccount, err := repos.SavingsAccounts.GetByID(ctx, input.SavingsAccountID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	initialRequest, err := repos.StudentLeavingRequests.GetByID(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if initialAccount.AcademicYearID == uuid.Nil || initialRequest.AcademicYearID == uuid.Nil {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidID
	}

	// Semua tahun dikunci sebelum rekening agar akun lintas tahun tetap mengikuti urutan global.
	yearIDs := []uuid.UUID{initialAccount.AcademicYearID, initialRequest.AcademicYearID}
	if yearIDs[0] == yearIDs[1] {
		yearIDs = yearIDs[:1]
	} else {
		sort.Slice(yearIDs, func(i, j int) bool {
			return yearIDs[i].String() < yearIDs[j].String()
		})
	}
	lockedYears := make(map[uuid.UUID]domain.AcademicYear, len(yearIDs))
	for _, yearID := range yearIDs {
		year, lockErr := repos.AcademicYears.GetByIDForShare(ctx, yearID)
		if lockErr != nil {
			return SettleStudentLeavingOutput{}, lockErr
		}
		lockedYears[year.ID] = year
	}
	accountYear := lockedYears[initialAccount.AcademicYearID]
	if accountYear.IsClosed() {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}

	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, input.SavingsAccountID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if account.StudentID != initialAccount.StudentID || account.AcademicYearID != initialAccount.AcademicYearID {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}

	student, err := repos.Students.GetByIDForUpdate(ctx, account.StudentID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if student.ID != account.StudentID || !student.IsActive() {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}

	request, err := repos.StudentLeavingRequests.GetByIDForUpdate(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if request.ID != initialRequest.ID ||
		request.StudentID != initialRequest.StudentID ||
		request.AcademicYearID != initialRequest.AcademicYearID ||
		request.StudentID != student.ID ||
		request.Status != domain.StudentLeavingInProgress {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}
	if !account.IsOpen() {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}
	if err := authorizeResponsibleWali(ctx, repos, input.Actor, account, accountYear); err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if pending, checkErr := hasPendingLateRequest(ctx, repos, account.ID); checkErr != nil {
		return SettleStudentLeavingOutput{}, checkErr
	} else if pending {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}
	issues, err := repos.StudentLeavingRequests.ListFinancialLinkIssues(ctx, request.ID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if len(issues) != 0 {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}

	priorSettlements, err := repos.Settlements.ListBySavingsAccount(ctx, account.ID, repository.ListOptions{Limit: 1})
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if len(priorSettlements) != 0 {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}
	if _, err := repos.Settlements.GetActiveReopenRequest(ctx, account.ID); err == nil {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	} else if !errors.Is(err, repository.ErrNotFound) {
		return SettleStudentLeavingOutput{}, err
	}

	transactions, err := repos.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	balance, err := domain.CalculateBalance(transactions)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if balance.Amount != input.ExpectedBalance {
		return SettleStudentLeavingOutput{}, repository.ErrConflict
	}

	settlement, err := domain.NewSavingsSettlement(
		account.ID,
		domain.SettlementStudentLeaving,
		balance.Amount,
		input.Actor.UserID,
		time.Now(),
		nil,
	)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	beforeAccount := account
	if err := account.Settle(settlement.ExecutedAt); err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if err := repos.SavingsAccounts.Update(ctx, account); err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if err := repos.Settlements.CreateForStudentLeavingRequest(ctx, settlement, request.ID); err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if err := s.createLeavingRequestSettlementAudit(ctx, repos, input.Actor, settlement, request.ID); err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if err := s.createSavingsAccountAudit(ctx, repos, input.Actor, "SETTLE", account, &beforeAccount); err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return SettleStudentLeavingOutput{}, fmt.Errorf("commit leaving request settlement: %w", err)
		}
		committed = true
	}

	return SettleStudentLeavingOutput{Settlement: &settlement, Account: &account, Student: &student}, nil
}

func (s *settlementService) createLeavingRequestSettlementAudit(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	settlement domain.SavingsSettlement,
	requestID uuid.UUID,
) error {
	data, err := json.Marshal(struct {
		Settlement              domain.SavingsSettlement `json:"settlement"`
		StudentLeavingRequestID uuid.UUID                `json:"student_leaving_request_id"`
	}{Settlement: settlement, StudentLeavingRequestID: requestID})
	if err != nil {
		return fmt.Errorf("marshal leaving request settlement audit: %w", err)
	}
	_, err = recordAudit(ctx, repos, RecordAuditInput{
		Actor:      actor,
		Action:     "SETTLE",
		EntityType: domain.AuditEntitySavingsSettlement,
		EntityID:   settlement.ID,
		After:      data,
	})
	return err
}
