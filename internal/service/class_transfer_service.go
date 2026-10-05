package service

import (
	"context"
	"strings"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ ClassTransferService = (*classTransferService)(nil)

func (s *classTransferService) Request(
	ctx context.Context,
	input RequestTransferInput,
) (RequestTransferOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return RequestTransferOutput{}, err
	}

	if input.StudentID == uuid.Nil ||
		input.AcademicYearID == uuid.Nil ||
		input.FromClassID == uuid.Nil ||
		input.ToClassID == uuid.Nil {
		return RequestTransferOutput{}, domain.ErrInvalidID
	}

	if input.FromClassID == input.ToClassID {
		return RequestTransferOutput{}, domain.ErrInvalidValue
	}

	if input.Actor.Role == domain.RoleSiswa {
		return RequestTransferOutput{}, ErrForbidden
	}

	if input.Actor.Role == domain.RoleWaliKelas {
		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			input.StudentID,
			input.AcademicYearID,
		); err != nil {
			return RequestTransferOutput{}, err
		}
	} else if input.Actor.Role != domain.RoleAdmin {
		return RequestTransferOutput{}, ErrForbidden
	}

	student, err := s.deps.Repositories.Students.GetByID(
		ctx,
		input.StudentID,
	)
	if err != nil {
		return RequestTransferOutput{}, err
	}

	if student.ID == uuid.Nil {
		return RequestTransferOutput{}, ErrInvalidDependency
	}

	academicYear, err := s.deps.Repositories.AcademicYears.GetByID(
		ctx,
		input.AcademicYearID,
	)
	if err != nil {
		return RequestTransferOutput{}, err
	}

	fromClass, err := s.deps.Repositories.Classes.GetByID(
		ctx,
		input.FromClassID,
	)
	if err != nil {
		return RequestTransferOutput{}, err
	}

	toClass, err := s.deps.Repositories.Classes.GetByID(
		ctx,
		input.ToClassID,
	)
	if err != nil {
		return RequestTransferOutput{}, err
	}

	if fromClass.ID == uuid.Nil || toClass.ID == uuid.Nil {
		return RequestTransferOutput{}, ErrInvalidDependency
	}

	currentHistory, err :=
		s.deps.Repositories.StudentClassHistories.
			GetCurrentByStudentAndAcademicYear(
				ctx,
				input.StudentID,
				input.AcademicYearID,
				time.Now(),
			)
	if err != nil {
		return RequestTransferOutput{}, err
	}

	if currentHistory.ClassID != input.FromClassID {
		return RequestTransferOutput{}, ErrInvalidDependency
	}

	if !currentHistory.Contains(time.Now()) {
		return RequestTransferOutput{}, ErrInvalidDependency
	}

	if !academicYear.StartDate.Before(academicYear.EndDate) {
		return RequestTransferOutput{}, domain.ErrInvalidDateRange
	}

	request, err := domain.NewClassTransferRequest(
		input.StudentID,
		input.AcademicYearID,
		input.FromClassID,
		input.ToClassID,
		input.Actor.UserID,
	)
	if err != nil {
		return RequestTransferOutput{}, err
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return RequestTransferOutput{}, err
	}

	rollback := true
	defer func() {
		if rollback {
			_ = uow.Rollback()
		}
	}()

	if err := uow.Repositories().ClassTransferRequests.Create(
		ctx,
		request,
	); err != nil {
		return RequestTransferOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return RequestTransferOutput{}, err
	}

	rollback = false

	return RequestTransferOutput{
		TransferRequest: classTransferRequestPointer(request),
	}, nil
}

func (s *classTransferService) Get(
	ctx context.Context,
	input GetTransferInput,
) (GetTransferOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetTransferOutput{}, err
	}

	if input.TransferRequestID == uuid.Nil {
		return GetTransferOutput{}, domain.ErrInvalidID
	}

	request, err :=
		s.deps.Repositories.ClassTransferRequests.GetByID(
			ctx,
			input.TransferRequestID,
		)
	if err != nil {
		return GetTransferOutput{}, err
	}

	if err := s.requireTransferScope(ctx, input.Actor, request); err != nil {
		return GetTransferOutput{}, err
	}

	return GetTransferOutput{
		TransferRequest: classTransferRequestPointer(request),
	}, nil
}

func (s *classTransferService) ListByStudent(
	ctx context.Context,
	input ListStudentTransfersInput,
) (ListStudentTransfersOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListStudentTransfersOutput{}, err
	}

	if input.StudentID == uuid.Nil ||
		input.AcademicYearID == uuid.Nil {
		return ListStudentTransfersOutput{}, domain.ErrInvalidID
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
	case domain.RoleSiswa:
		student, err := s.deps.Repositories.Students.GetByUserID(
			ctx,
			input.Actor.UserID,
		)
		if err != nil {
			return ListStudentTransfersOutput{}, err
		}

		if student.ID != input.StudentID {
			return ListStudentTransfersOutput{}, ErrScopeViolation
		}
	case domain.RoleWaliKelas:
		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			input.StudentID,
			input.AcademicYearID,
		); err != nil {
			return ListStudentTransfersOutput{}, err
		}
	default:
		return ListStudentTransfersOutput{}, ErrForbidden
	}

	requests, err :=
		s.deps.Repositories.ClassTransferRequests.
			ListByStudentAndAcademicYear(
				ctx,
				input.StudentID,
				input.AcademicYearID,
				repository.ListOptions{
					Limit:  input.Options.Limit,
					Offset: input.Options.Offset,
				},
			)
	if err != nil {
		return ListStudentTransfersOutput{}, err
	}

	return ListStudentTransfersOutput{
		TransferRequests: classTransferRequestPointers(requests),
	}, nil
}

func (s *classTransferService) ListPending(
	ctx context.Context,
	input ListPendingTransfersInput,
) (ListPendingTransfersOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListPendingTransfersOutput{}, err
	}

	if input.AcademicYearID == uuid.Nil {
		return ListPendingTransfersOutput{}, domain.ErrInvalidID
	}

	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ListPendingTransfersOutput{}, err
	}

	requests, err :=
		s.deps.Repositories.ClassTransferRequests.
			ListPending(
				ctx,
				input.AcademicYearID,
				repository.ListOptions{
					Limit:  input.Options.Limit,
					Offset: input.Options.Offset,
				},
			)
	if err != nil {
		return ListPendingTransfersOutput{}, err
	}

	return ListPendingTransfersOutput{
		TransferRequests: classTransferRequestPointers(requests),
	}, nil
}

func (s *classTransferService) Approve(
	ctx context.Context,
	input ApproveTransferInput,
) (ApproveTransferOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ApproveTransferOutput{}, err
	}

	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ApproveTransferOutput{}, err
	}

	if input.TransferRequestID == uuid.Nil {
		return ApproveTransferOutput{}, domain.ErrInvalidID
	}

	if input.EffectiveDate.IsZero() {
		return ApproveTransferOutput{}, domain.ErrInvalidValue
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return ApproveTransferOutput{}, err
	}

	rollback := true
	defer func() {
		if rollback {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()

	request, err :=
		repos.ClassTransferRequests.GetByIDForUpdate(
			ctx,
			input.TransferRequestID,
		)
	if err != nil {
		return ApproveTransferOutput{}, err
	}

	academicYear, err :=
		repos.AcademicYears.GetByID(
			ctx,
			request.AcademicYearID,
		)
	if err != nil {
		return ApproveTransferOutput{}, err
	}

	if input.EffectiveDate.Before(academicYear.StartDate) ||
		!input.EffectiveDate.Before(academicYear.EndDate) {
		return ApproveTransferOutput{}, domain.ErrInvalidDateRange
	}

	currentHistory, err :=
		repos.StudentClassHistories.
			GetCurrentByStudentAndAcademicYear(
				ctx,
				request.StudentID,
				request.AcademicYearID,
				input.EffectiveDate,
			)
	if err != nil {
		return ApproveTransferOutput{}, err
	}

	if currentHistory.ClassID != request.FromClassID {
		return ApproveTransferOutput{}, ErrInvalidDependency
	}

	if !currentHistory.Contains(input.EffectiveDate) {
		return ApproveTransferOutput{}, domain.ErrInvalidDateRange
	}

	if err := request.Approve(
		input.Actor.UserID,
		time.Now(),
	); err != nil {
		return ApproveTransferOutput{}, err
	}

	_, err =
		repos.StudentClassHistories.
			CloseCurrentByStudentAndAcademicYear(
				ctx,
				request.StudentID,
				request.AcademicYearID,
				input.EffectiveDate,
			)
	if err != nil {
		return ApproveTransferOutput{}, err
	}

	newHistory, err := domain.NewStudentClassHistory(
		request.StudentID,
		request.AcademicYearID,
		request.ToClassID,
		domain.DateRange{
			From: input.EffectiveDate,
			To:   nil,
		},
	)
	if err != nil {
		return ApproveTransferOutput{}, err
	}

	if err := repos.StudentClassHistories.Create(
		ctx,
		newHistory,
	); err != nil {
		return ApproveTransferOutput{}, err
	}

	if err := repos.ClassTransferRequests.Update(
		ctx,
		request,
	); err != nil {
		return ApproveTransferOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return ApproveTransferOutput{}, err
	}

	rollback = false

	return ApproveTransferOutput{
		TransferRequest: classTransferRequestPointer(request),
		ClassHistory:    classHistoryPointer(newHistory),
	}, nil
}

func (s *classTransferService) Reject(
	ctx context.Context,
	input RejectTransferInput,
) (RejectTransferOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return RejectTransferOutput{}, err
	}

	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return RejectTransferOutput{}, err
	}

	if input.TransferRequestID == uuid.Nil {
		return RejectTransferOutput{}, domain.ErrInvalidID
	}

	if strings.TrimSpace(input.RejectionReason) == "" {
		return RejectTransferOutput{}, domain.ErrEmptyRejectionReason
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return RejectTransferOutput{}, err
	}

	rollback := true
	defer func() {
		if rollback {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()

	request, err :=
		repos.ClassTransferRequests.GetByIDForUpdate(
			ctx,
			input.TransferRequestID,
		)
	if err != nil {
		return RejectTransferOutput{}, err
	}

	if err := request.Reject(
		input.Actor.UserID,
		time.Now(),
		input.RejectionReason,
	); err != nil {
		return RejectTransferOutput{}, err
	}

	if err := repos.ClassTransferRequests.Update(
		ctx,
		request,
	); err != nil {
		return RejectTransferOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return RejectTransferOutput{}, err
	}

	rollback = false

	return RejectTransferOutput{
		TransferRequest: classTransferRequestPointer(request),
	}, nil
}

func (s *classTransferService) requireTransferScope(
	ctx context.Context,
	actor Actor,
	request domain.ClassTransferRequest,
) error {
	switch actor.Role {
	case domain.RoleAdmin:
		return nil

	case domain.RoleSiswa:
		student, err := s.deps.Repositories.Students.GetByUserID(
			ctx,
			actor.UserID,
		)
		if err != nil {
			return err
		}

		if student.ID != request.StudentID {
			return ErrScopeViolation
		}

		return nil

	case domain.RoleWaliKelas:
		return RequireStudentScope(
			ctx,
			s.deps.Repositories,
			actor,
			request.StudentID,
			request.AcademicYearID,
		)

	default:
		return ErrForbidden
	}
}

func classTransferRequestPointer(
	request domain.ClassTransferRequest,
) *domain.ClassTransferRequest {
	return &request
}

func classTransferRequestPointers(
	requests []domain.ClassTransferRequest,
) []*domain.ClassTransferRequest {
	output := make([]*domain.ClassTransferRequest, 0, len(requests))

	for _, request := range requests {
		requestCopy := request
		output = append(output, &requestCopy)
	}

	return output
}

func classHistoryPointer(
	history domain.StudentClassHistory,
) *domain.StudentClassHistory {
	return &history
}
