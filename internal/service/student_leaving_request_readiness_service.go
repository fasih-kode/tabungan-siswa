package service

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func (s *studentLeavingRequestService) Readiness(
	ctx context.Context,
	input StudentLeavingReadinessInput,
) (StudentLeavingReadinessOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	if input.StudentLeavingRequestID == uuid.Nil {
		return StudentLeavingReadinessOutput{}, domain.ErrInvalidID
	}
	manager, ok := s.deps.UOW.(repository.ReadOnlySnapshotUnitOfWorkManager)
	if !ok {
		return StudentLeavingReadinessOutput{}, ErrInvalidDependency
	}
	uow, err := manager.BeginReadOnlySnapshot(ctx)
	if err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()
	repos := uow.Repositories()
	if repos.StudentLeavingRequests == nil || repos.Students == nil || repos.AcademicYears == nil ||
		repos.SavingsAccounts == nil || repos.Settlements == nil || repos.Transactions == nil ||
		repos.LeavingTransactionRequests == nil {
		return StudentLeavingReadinessOutput{}, ErrInvalidDependency
	}

	request, err := repos.StudentLeavingRequests.GetByID(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	student, err := repos.Students.GetByID(ctx, request.StudentID)
	if err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	output, err := s.studentLeavingReadiness(ctx, repos, request, student)
	if err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	if err := uow.Commit(); err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	committed = true
	return output, nil
}

// studentLeavingReadiness memakai repository UOW pemanggil agar validasi dan mutasi melihat transaksi yang sama.
func (s *studentLeavingRequestService) studentLeavingReadiness(ctx context.Context, repos repository.RepositorySet, request domain.StudentLeavingRequest, student domain.Student) (StudentLeavingReadinessOutput, error) {
	if repos.AcademicYears == nil || repos.SavingsAccounts == nil || repos.Settlements == nil || repos.Transactions == nil || repos.LeavingTransactionRequests == nil || repos.StudentLeavingRequests == nil {
		return StudentLeavingReadinessOutput{}, ErrInvalidDependency
	}

	output := StudentLeavingReadinessOutput{
		Request:  request,
		Student:  student,
		Blockers: make([]StudentLeavingReadinessBlocker, 0),
	}
	if request.Status != domain.StudentLeavingInProgress {
		output.addBlocker(LeavingBlockerProcessNotInProgress, nil, &request.ID,
			"Proses pindah tidak lagi berjalan.")
	}
	if !student.IsActive() {
		output.addBlocker(LeavingBlockerStudentNotActive, nil, &student.ID,
			"Status siswa bukan aktif.")
	}
	referenceYear, err := repos.AcademicYears.GetByID(ctx, request.AcademicYearID)
	if err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	today, err := jakartaCalendarDate(s.now())
	if err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	if calendarDateAfter(request.EffectiveLeavingDate, today) || !calendarDateWithin(request.EffectiveLeavingDate, referenceYear.StartDate, referenceYear.EndDate) {
		output.addBlocker(LeavingBlockerReferenceYearInvalid, nil, &request.ID,
			"Tanggal efektif pindah berada di luar tahun ajaran referensi atau masih mendatang.")
	}

	accounts, err := repos.SavingsAccounts.ListByStudent(ctx, student.ID)
	if err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	output.TotalAccounts = len(accounts)
	for _, account := range accounts {
		if account.ID == uuid.Nil || account.StudentID != student.ID || account.AcademicYearID == uuid.Nil {
			output.addBlocker(LeavingBlockerSettlementInvalid, &account.ID, nil,
				"Data rekening tidak konsisten dengan siswa.")
			continue
		}
		if err := s.checkLeavingAccountReadiness(ctx, repos, request, account, &output); err != nil {
			return StudentLeavingReadinessOutput{}, err
		}
	}

	issues, err := repos.StudentLeavingRequests.ListFinancialLinkIssues(ctx, request.ID)
	if err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	for _, issue := range issues {
		code, message := leavingLinkIssueBlocker(issue.Type)
		output.addBlocker(code, nil, &issue.RecordID, message)
	}
	linkedSettlements, err := repos.Settlements.ListByStudentLeavingRequest(ctx, request.ID)
	if err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	for _, settlement := range linkedSettlements {
		account, ok := findReadinessAccount(accounts, settlement.SavingsAccountID)
		if !ok || settlement.ID == uuid.Nil || settlement.Type != domain.SettlementStudentLeaving ||
			(settlement.Status != domain.SettlementCompleted && settlement.Status != domain.SettlementSuperseded) {
			output.addBlocker(LeavingBlockerFinancialLinkInconsistent, nil, &settlement.ID,
				"Settlement yang tertaut tidak sesuai dengan rekening siswa.")
			continue
		}
		if settlement.Status == domain.SettlementSuperseded {
			continue
		}
		current, currentErr := repos.Settlements.GetCompletedBySavingsAccount(ctx, account.ID)
		if errors.Is(currentErr, repository.ErrNotFound) || (currentErr == nil && current.ID != settlement.ID) {
			output.addBlocker(LeavingBlockerSettlementInvalid, &account.ID, &settlement.ID,
				"Settlement proses pindah bukan settlement aktif rekening.")
		} else if currentErr != nil {
			return StudentLeavingReadinessOutput{}, currentErr
		}
	}
	if err := checkLeavingPlacement(ctx, repos, request, &output); err != nil {
		return StudentLeavingReadinessOutput{}, err
	}
	output.Ready = len(output.Blockers) == 0
	return output, nil
}

func checkLeavingPlacement(ctx context.Context, repos repository.RepositorySet, request domain.StudentLeavingRequest, output *StudentLeavingReadinessOutput) error {
	if repos.StudentClassHistories == nil || repos.ClassTransferRequests == nil {
		return ErrInvalidDependency
	}
	history, err := repos.StudentClassHistories.GetCurrentByStudentAndAcademicYear(ctx, request.StudentID, request.AcademicYearID, request.EffectiveLeavingDate)
	if errors.Is(err, repository.ErrNotFound) {
		output.addBlocker(LeavingBlockerPlacementInvalid, nil, &request.ID, "Penempatan kelas pada tanggal efektif pindah tidak ditemukan.")
	} else if err != nil {
		return err
	} else if history.StudentID != request.StudentID || history.AcademicYearID != request.AcademicYearID || !history.Contains(request.EffectiveLeavingDate) || !calendarDateAfter(request.EffectiveLeavingDate, history.Period.From) {
		output.addBlocker(LeavingBlockerPlacementInvalid, nil, &history.ID, "Penempatan tidak dapat ditutup pada tanggal efektif; tanggal pindah harus setelah awal penempatan.")
	}
	later, err := repos.StudentClassHistories.HasPlacementStartingAfter(ctx, request.StudentID, request.EffectiveLeavingDate)
	if err != nil {
		return err
	}
	if later {
		output.addBlocker(LeavingBlockerLaterPlacement, nil, &request.ID, "Ada penempatan kelas setelah tanggal efektif pindah yang perlu ditinjau.")
	}
	pending, err := repos.ClassTransferRequests.HasPendingByStudent(ctx, request.StudentID)
	if err != nil {
		return err
	}
	if pending {
		output.addBlocker(LeavingBlockerPendingTransfer, nil, &request.ID, "Permintaan pindah kelas masih menunggu keputusan.")
	}
	return nil
}

func leavingLinkIssueBlocker(issue repository.StudentLeavingFinancialLinkIssueType) (StudentLeavingReadinessBlockerCode, string) {
	switch issue {
	case repository.StudentLeavingSettlementStudentMismatch:
		return LeavingBlockerSettlementStudentMismatch, "Settlement terhubung ke siswa yang berbeda."
	case repository.StudentLeavingSettlementTypeMismatch:
		return LeavingBlockerSettlementTypeMismatch, "Settlement yang terhubung bukan settlement pindah sekolah."
	case repository.StudentLeavingLateRequestStudentMismatch:
		return LeavingBlockerLateRequestStudentMismatch, "Permintaan transaksi terlambat terhubung ke siswa yang berbeda."
	case repository.StudentLeavingLateRequestYearMismatch:
		return LeavingBlockerLinkedLateRequestYearMismatch, "Tahun ajaran permintaan terlambat terhubung tidak sesuai dengan rekening."
	default:
		return LeavingBlockerFinancialLinkInconsistent, "Relasi data keuangan pada proses pindah tidak konsisten."
	}
}

func (s *studentLeavingRequestService) checkLeavingAccountReadiness(
	ctx context.Context,
	repos repository.RepositorySet,
	request domain.StudentLeavingRequest,
	account domain.SavingsAccount,
	output *StudentLeavingReadinessOutput,
) error {
	var settlementForBalance *domain.SavingsSettlement
	if account.IsOpen() {
		output.OpenAccounts++
		year, err := repos.AcademicYears.GetByID(ctx, account.AcademicYearID)
		if err != nil {
			return err
		}
		code, message := LeavingBlockerAccountOpen, "Rekening masih terbuka dan perlu diselesaikan."
		if year.IsClosed() {
			code = LeavingBlockerAccountOpenYearClosed
			message = "Rekening masih terbuka pada tahun ajaran yang sudah ditutup."
		}
		output.addBlocker(code, &account.ID, nil, message)
	} else if account.IsSettled() {
		output.SettledAccounts++
		if account.SettledAt == nil || account.SettledAt.IsZero() {
			output.addBlocker(LeavingBlockerSettlementInvalid, &account.ID, nil,
				"Rekening berstatus selesai tanpa waktu penyelesaian yang sah.")
		}
		settlement, err := repos.Settlements.GetCompletedBySavingsAccount(ctx, account.ID)
		if errors.Is(err, repository.ErrNotFound) {
			output.addBlocker(LeavingBlockerSettlementMissing, &account.ID, nil,
				"Rekening selesai tidak memiliki settlement yang masih berlaku.")
		} else if err != nil {
			return err
		} else if !validReadinessSettlement(settlement, account.ID) {
			output.addBlocker(LeavingBlockerSettlementInvalid, &account.ID, &settlement.ID,
				"Settlement rekening tidak memiliki data yang sah.")
		} else {
			settlementForBalance = &settlement
			if err := s.checkLeavingSettlementHandover(ctx, repos, settlement, account.ID, output); err != nil {
				return err
			}
		}
	} else {
		output.addBlocker(LeavingBlockerSettlementInvalid, &account.ID, nil,
			"Status rekening tidak dikenal.")
	}

	requests, err := repos.LeavingTransactionRequests.ListBySavingsAccount(ctx, account.ID)
	if err != nil {
		return err
	}
	transactions, err := repos.Transactions.ListAllBySavingsAccount(ctx, account.ID)
	if err != nil {
		return err
	}
	if account.IsSettled() {
		balance, balanceErr := domain.CalculateBalance(transactions)
		if balanceErr != nil {
			output.addBlocker(LeavingBlockerLedgerInvalid, &account.ID, readinessSettlementID(settlementForBalance),
				"Transaksi rekening tidak dapat dihitung sebagai saldo yang sah.")
		} else if settlementForBalance != nil && balance.Amount != settlementForBalance.Amount {
			output.addBlocker(LeavingBlockerSettlementLedgerMismatch, &account.ID, &settlementForBalance.ID,
				"Nominal penyelesaian tabungan tidak sesuai dengan saldo transaksi rekening.")
		}
	}
	transactionsByLateRequest := make(map[uuid.UUID]domain.Transaction)
	lateRequestsByID := make(map[uuid.UUID]domain.LeavingTransactionRequest, len(requests))
	for _, lateRequest := range requests {
		lateRequestsByID[lateRequest.ID] = lateRequest
	}
	for _, transaction := range transactions {
		if transaction.LateRequestID != nil {
			transactionsByLateRequest[*transaction.LateRequestID] = transaction
			lateRequest, ok := lateRequestsByID[*transaction.LateRequestID]
			if !ok || lateRequest.Status != domain.LeavingTransactionCompleted ||
				lateRequest.Type != transaction.Type || lateRequest.Amount != transaction.Amount ||
				!sameCalendarDate(lateRequest.TransactionDate, transaction.TransactionDate) {
				output.addBlocker(LeavingBlockerTransactionLateLinkInvalid, &account.ID, &transaction.ID,
					"Relasi transaksi ledger ke permintaan terlambat tidak konsisten.")
			}
		}
	}
	blockingLateRequestIDs := make(map[uuid.UUID]struct{})
	for _, lateRequest := range requests {
		if lateRequest.SavingsAccountID != account.ID || lateRequest.AcademicYearID != account.AcademicYearID {
			blockingLateRequestIDs[lateRequest.ID] = struct{}{}
			output.addBlocker(LeavingBlockerLateRequestYearMismatch, &account.ID, &lateRequest.ID,
				"Tahun ajaran permintaan transaksi terlambat tidak sesuai dengan rekening.")
		}
		switch lateRequest.Status {
		case domain.LeavingTransactionPending:
			blockingLateRequestIDs[lateRequest.ID] = struct{}{}
			output.addBlocker(LeavingBlockerLateRequestPending, &account.ID, &lateRequest.ID,
				"Masih ada permintaan transaksi terlambat yang menunggu keputusan.")
		case domain.LeavingTransactionApproved:
			blockingLateRequestIDs[lateRequest.ID] = struct{}{}
			output.addBlocker(LeavingBlockerLateRequestApproved, &account.ID, &lateRequest.ID,
				"Masih ada permintaan transaksi terlambat yang disetujui tetapi belum dicatat.")
		case domain.LeavingTransactionCompleted:
			transaction, ok := transactionsByLateRequest[lateRequest.ID]
			if !ok {
				blockingLateRequestIDs[lateRequest.ID] = struct{}{}
				output.addBlocker(LeavingBlockerLateRequestLedgerMissing, &account.ID, &lateRequest.ID,
					"Permintaan transaksi terlambat selesai tetapi transaksi ledger tidak ditemukan.")
			} else if transaction.Status != domain.TransactionActive || transaction.SavingsAccountID != account.ID ||
				transaction.Type != lateRequest.Type || transaction.Amount != lateRequest.Amount ||
				!sameCalendarDate(transaction.TransactionDate, lateRequest.TransactionDate) {
				blockingLateRequestIDs[lateRequest.ID] = struct{}{}
				output.addBlocker(LeavingBlockerLateRequestLedgerInvalid, &account.ID, &lateRequest.ID,
					"Transaksi ledger permintaan terlambat tidak sesuai dengan rincian yang disetujui.")
			}
		}
	}
	output.BlockingLateRequests += len(blockingLateRequestIDs)
	for _, transaction := range transactions {
		if transaction.Status == domain.TransactionActive && calendarDateAfter(transaction.TransactionDate, request.EffectiveLeavingDate) {
			output.addBlocker(LeavingBlockerTransactionAfterLeavingDate, &account.ID, &transaction.ID,
				"Ada transaksi aktif setelah tanggal efektif pindah yang perlu ditinjau.")
		}
	}
	return nil
}

func readinessSettlementID(settlement *domain.SavingsSettlement) *uuid.UUID {
	if settlement == nil {
		return nil
	}
	return &settlement.ID
}

func validReadinessSettlement(settlement domain.SavingsSettlement, accountID uuid.UUID) bool {
	return settlement.ID != uuid.Nil && settlement.SavingsAccountID == accountID &&
		settlement.Type.IsValid() && settlement.Status == domain.SettlementCompleted &&
		settlement.ExecutedBy != uuid.Nil && !settlement.ExecutedAt.IsZero() &&
		settlement.Amount >= 0 && settlement.Amount <= domain.MaxMoney
}

func (s *studentLeavingRequestService) checkLeavingSettlementHandover(
	ctx context.Context,
	repos repository.RepositorySet,
	settlement domain.SavingsSettlement,
	accountID uuid.UUID,
	output *StudentLeavingReadinessOutput,
) error {
	movements, err := repos.Settlements.ListCashMovements(ctx, settlement.ID)
	if err != nil {
		return err
	}
	net, valid := readinessHandoverNet(settlement.ID, movements)
	if !valid || net < 0 || net > settlement.Amount {
		output.addBlocker(LeavingBlockerHandoverInvalid, &accountID, &settlement.ID,
			"Catatan penyerahan settlement tidak konsisten dengan nominal kewajiban.")
		return nil
	}
	if settlement.Amount == 0 && len(movements) != 0 {
		output.addBlocker(LeavingBlockerHandoverInvalid, &accountID, &settlement.ID,
			"Settlement bernilai nol tidak seharusnya memiliki pergerakan uang fisik.")
		return nil
	}
	if net < settlement.Amount {
		output.OutstandingHandovers++
		output.addBlocker(LeavingBlockerHandoverIncomplete, &accountID, &settlement.ID,
			"Penyerahan uang settlement belum lengkap.")
	}
	return nil
}

func findReadinessAccount(accounts []domain.SavingsAccount, accountID uuid.UUID) (domain.SavingsAccount, bool) {
	for _, account := range accounts {
		if account.ID == accountID {
			return account, true
		}
	}
	return domain.SavingsAccount{}, false
}

func calendarDateWithin(value, start, end time.Time) bool {
	return !calendarDateAfter(start, value) && calendarDateAfter(end, value)
}

func sameCalendarDate(left, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}

func readinessHandoverNet(
	settlementID uuid.UUID,
	movements []domain.SettlementCashMovement,
) (domain.Money, bool) {
	var net domain.Money
	for _, movement := range movements {
		if movement.ID == uuid.Nil || movement.SavingsSettlementID != settlementID ||
			movement.ConfirmedBy == uuid.Nil || movement.ConfirmedAt.IsZero() ||
			movement.Amount <= 0 || movement.Amount > domain.MaxMoney {
			return 0, false
		}
		switch movement.Kind {
		case domain.SettlementCashHandover:
			if int64(net) > math.MaxInt64-int64(movement.Amount) {
				return 0, false
			}
			net += movement.Amount
		case domain.SettlementCashReturn:
			if int64(net) < math.MinInt64+int64(movement.Amount) {
				return 0, false
			}
			net -= movement.Amount
		default:
			return 0, false
		}
	}
	return net, true
}

func calendarDateAfter(left, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	if leftYear != rightYear {
		return leftYear > rightYear
	}
	if leftMonth != rightMonth {
		return leftMonth > rightMonth
	}
	return leftDay > rightDay
}

func (output *StudentLeavingReadinessOutput) addBlocker(
	code StudentLeavingReadinessBlockerCode,
	accountID, recordID *uuid.UUID,
	message string,
) {
	output.Blockers = append(output.Blockers, StudentLeavingReadinessBlocker{
		Code: code, AccountID: accountID, RecordID: recordID, Message: message,
	})
	sort.Slice(output.Blockers, func(i, j int) bool {
		left, right := output.Blockers[i], output.Blockers[j]
		leftAccount, rightAccount := "", ""
		if left.AccountID != nil {
			leftAccount = left.AccountID.String()
		}
		if right.AccountID != nil {
			rightAccount = right.AccountID.String()
		}
		if leftAccount != rightAccount {
			return leftAccount < rightAccount
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		leftRecord, rightRecord := "", ""
		if left.RecordID != nil {
			leftRecord = left.RecordID.String()
		}
		if right.RecordID != nil {
			rightRecord = right.RecordID.String()
		}
		return leftRecord < rightRecord
	})
}
