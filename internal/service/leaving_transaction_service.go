package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type leavingTransactionRequestService struct{ deps Dependencies }

func NewLeavingTransactionRequestService(deps Dependencies) (LeavingTransactionRequestService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}
	return &leavingTransactionRequestService{deps: deps}, nil
}

func (s *leavingTransactionRequestService) Create(ctx context.Context, in CreateLeavingTransactionRequestInput) (LeavingTransactionRequestOutput, error) {
	if err := RequireRole(in.Actor, domain.RoleWaliKelas); err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	uow, repos, txCtx, owns, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	ctx = txCtx
	committed := false
	defer func() {
		if owns && !committed {
			_ = uow.Rollback()
		}
	}()
	unlocked, err := repos.SavingsAccounts.GetByID(ctx, in.SavingsAccountID)
	if err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	year, err := repos.AcademicYears.GetByIDForShare(ctx, unlocked.AcademicYearID)
	if err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	if year.IsClosed() {
		return LeavingTransactionRequestOutput{}, domain.ErrInvalidTransition
	}
	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, in.SavingsAccountID)
	if err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	student, err := repos.Students.GetByIDForUpdate(ctx, account.StudentID)
	if err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	if !account.IsOpen() || year.IsClosed() || !student.IsLeft() || student.LeavingDate == nil {
		return LeavingTransactionRequestOutput{}, domain.ErrInvalidTransition
	}
	if err := RequireStudentScope(ctx, repos, in.Actor, student.ID, year.ID); err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	d := dateOnly(in.TransactionDate)
	leave := dateOnly(*student.LeavingDate)
	if d.Before(dateOnly(year.StartDate)) || !d.Before(dateOnly(year.EndDate)) || !d.Before(leave) {
		return LeavingTransactionRequestOutput{}, domain.ErrInvalidDateRange
	}
	x, err := domain.NewLeavingTransactionRequest(account.ID, year.ID, in.Type, in.Amount, d, in.Reason, in.EvidenceReference, in.Actor.UserID)
	if err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	if repos.LeavingTransactionRequests == nil {
		return LeavingTransactionRequestOutput{}, ErrInvalidDependency
	}
	usedEvidence, err := repos.LeavingTransactionRequests.HasEvidenceReference(ctx, account.ID, x.EvidenceReference)
	if err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	if usedEvidence {
		return LeavingTransactionRequestOutput{}, repository.ErrConflict
	}
	if err := repos.LeavingTransactionRequests.CreateLeavingTransactionRequest(ctx, x); err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	if err := s.audit(ctx, repos, in.Actor, "REQUEST", x, nil); err != nil {
		return LeavingTransactionRequestOutput{}, err
	}
	if owns {
		if err := uow.Commit(); err != nil {
			return LeavingTransactionRequestOutput{}, err
		}
		committed = true
	}
	return LeavingTransactionRequestOutput{Request: x, Account: account, Student: student, Year: year}, nil
}

func (s *leavingTransactionRequestService) List(ctx context.Context, in ListLeavingTransactionRequestsInput) (ListLeavingTransactionRequestsOutput, error) {
	if err := RequireRole(in.Actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		return ListLeavingTransactionRequestsOutput{}, err
	}
	if s.deps.Repositories.LeavingTransactionRequests == nil {
		return ListLeavingTransactionRequestsOutput{}, ErrInvalidDependency
	}
	rows, err := s.deps.Repositories.LeavingTransactionRequests.ListLeavingTransactionRequests(ctx, in.AcademicYearID, in.Status)
	if err != nil {
		return ListLeavingTransactionRequestsOutput{}, err
	}
	out := ListLeavingTransactionRequestsOutput{Rows: []LeavingTransactionRequestOutput{}}
	for _, x := range rows {
		a, e := s.deps.Repositories.SavingsAccounts.GetByID(ctx, x.SavingsAccountID)
		if e != nil {
			return out, e
		}
		y, e := s.deps.Repositories.AcademicYears.GetByID(ctx, x.AcademicYearID)
		if e != nil {
			return out, e
		}
		st, e := s.deps.Repositories.Students.GetByID(ctx, a.StudentID)
		if e != nil {
			return out, e
		}
		if in.Actor.Role == domain.RoleWaliKelas {
			if e = RequireStudentScope(ctx, s.deps.Repositories, in.Actor, st.ID, y.ID); e != nil {
				if errors.Is(e, ErrScopeViolation) {
					continue
				}
				return out, e
			}
		}
		out.Rows = append(out.Rows, LeavingTransactionRequestOutput{Request: x, Account: a, Student: st, Year: y})
	}
	return out, nil
}

func (s *leavingTransactionRequestService) Decide(ctx context.Context, in DecideLeavingTransactionRequestInput) error {
	if err := RequireRole(in.Actor, domain.RoleAdmin); err != nil {
		return err
	}
	if in.RequestID == uuid.Nil {
		return domain.ErrInvalidID
	}
	uow, repos, txCtx, owns, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return err
	}
	ctx = txCtx
	committed := false
	defer func() {
		if owns && !committed {
			_ = uow.Rollback()
		}
	}()
	x, err := repos.LeavingTransactionRequests.GetLeavingTransactionRequestForUpdate(ctx, in.RequestID)
	if err != nil {
		return err
	}
	before := x
	if err := x.Decide(in.Actor.UserID, in.Approve, in.Reason); err != nil {
		return err
	}
	if err := repos.LeavingTransactionRequests.UpdateLeavingTransactionRequest(ctx, x); err != nil {
		return err
	}
	if err := s.audit(ctx, repos, in.Actor, "DECIDE", x, &before); err != nil {
		return err
	}
	if owns {
		if err := uow.Commit(); err != nil {
			return err
		}
		committed = true
	}
	return nil
}

func (s *leavingTransactionRequestService) Execute(ctx context.Context, in ExecuteLeavingTransactionRequestInput) (domain.Transaction, error) {
	if err := RequireRole(in.Actor, domain.RoleWaliKelas); err != nil {
		return domain.Transaction{}, err
	}
	if in.RequestID == uuid.Nil {
		return domain.Transaction{}, domain.ErrInvalidID
	}
	uow, repos, txCtx, owns, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return domain.Transaction{}, err
	}
	ctx = txCtx
	committed := false
	defer func() {
		if owns && !committed {
			_ = uow.Rollback()
		}
	}()
	initial, err := repos.LeavingTransactionRequests.GetLeavingTransactionRequest(ctx, in.RequestID)
	if err != nil {
		return domain.Transaction{}, err
	}
	year, err := repos.AcademicYears.GetByIDForShare(ctx, initial.AcademicYearID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if year.IsClosed() {
		return domain.Transaction{}, domain.ErrInvalidTransition
	}
	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, initial.SavingsAccountID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if !account.IsOpen() || account.AcademicYearID != year.ID {
		return domain.Transaction{}, domain.ErrInvalidTransition
	}
	student, err := repos.Students.GetByIDForUpdate(ctx, account.StudentID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if !student.IsLeft() || student.LeavingDate == nil {
		return domain.Transaction{}, domain.ErrInvalidTransition
	}
	if err := RequireStudentScope(ctx, repos, in.Actor, student.ID, year.ID); err != nil {
		return domain.Transaction{}, err
	}
	x, err := repos.LeavingTransactionRequests.GetLeavingTransactionRequestForUpdate(ctx, in.RequestID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if x.Status != domain.LeavingTransactionApproved || x.SavingsAccountID != account.ID || x.AcademicYearID != year.ID {
		return domain.Transaction{}, domain.ErrInvalidTransition
	}
	if !x.TransactionDate.Before(dateOnly(*student.LeavingDate)) {
		return domain.Transaction{}, domain.ErrInvalidDateRange
	}
	all, err := repos.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
	if err != nil {
		return domain.Transaction{}, err
	}
	tx, err := domain.NewTransaction(account.ID, x.Type, x.Amount, x.TransactionDate, in.Actor.UserID)
	if err != nil {
		return domain.Transaction{}, err
	}
	reqID := x.ID
	tx.LateRequestID = &reqID
	all = append(all, tx)
	sortTransactionsChronologically(all)
	var running domain.Money
	for _, entry := range all {
		if entry.Type == domain.TransactionDeposit {
			running, err = running.Add(entry.Amount)
		} else {
			running, err = running.Subtract(entry.Amount)
		}
		if err != nil {
			return domain.Transaction{}, err
		}
		if running < 0 {
			return domain.Transaction{}, domain.ErrNegativeBalance
		}
	}
	if err := repos.Transactions.Create(ctx, tx); err != nil {
		return domain.Transaction{}, err
	}
	before := x
	if err := x.Complete(); err != nil {
		return domain.Transaction{}, err
	}
	if err := repos.LeavingTransactionRequests.UpdateLeavingTransactionRequest(ctx, x); err != nil {
		return domain.Transaction{}, err
	}
	if err := s.audit(ctx, repos, in.Actor, "EXECUTE", x, &before); err != nil {
		return domain.Transaction{}, err
	}
	if err := (&transactionService{deps: s.deps}).createTransactionAudit(ctx, repos, in.Actor, "CREATE_LATE_APPROVED", tx, nil); err != nil {
		return domain.Transaction{}, err
	}
	if owns {
		if err := uow.Commit(); err != nil {
			return domain.Transaction{}, err
		}
		committed = true
	}
	return tx, nil
}

func (s *leavingTransactionRequestService) audit(ctx context.Context, repos repository.RepositorySet, actor Actor, action string, x domain.LeavingTransactionRequest, before *domain.LeavingTransactionRequest) error {
	var b []byte
	var err error
	if before != nil {
		b, err = json.Marshal(before)
		if err != nil {
			return err
		}
	}
	a, err := json.Marshal(x)
	if err != nil {
		return err
	}
	_, err = recordAudit(ctx, repos, RecordAuditInput{Actor: actor, Action: action, EntityType: domain.AuditEntityLeavingTransactionRequest, EntityID: x.ID, Before: b, After: a})
	return err
}

func dateOnly(v time.Time) time.Time {
	return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.UTC)
}
