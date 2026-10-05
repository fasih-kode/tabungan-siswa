package service

import (
	"context"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type academicYearService struct {
	deps Dependencies
}

var _ AcademicYearService = (*academicYearService)(nil)

func NewAcademicYearService(deps Dependencies) (*academicYearService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &academicYearService{
		deps: deps,
	}, nil
}

func (s *academicYearService) Create(
	ctx context.Context,
	input CreateAcademicYearInput,
) (CreateAcademicYearOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return CreateAcademicYearOutput{}, err
	}

	academicYear, err := domain.NewAcademicYear(
		input.Name,
		input.StartDate,
		input.EndDate,
	)
	if err != nil {
		return CreateAcademicYearOutput{}, err
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return CreateAcademicYearOutput{}, fmt.Errorf(
			"begin create academic year transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	if err := uow.Repositories().AcademicYears.Create(ctx, academicYear); err != nil {
		return CreateAcademicYearOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return CreateAcademicYearOutput{}, fmt.Errorf(
			"commit create academic year transaction: %w",
			err,
		)
	}

	committed = true

	return CreateAcademicYearOutput{
		AcademicYear: &academicYear,
	}, nil
}

func (s *academicYearService) Get(
	ctx context.Context,
	input GetAcademicYearInput,
) (GetAcademicYearOutput, error) {
	if err := RequireRole(
		input.Actor,
		domain.RoleAdmin,
		domain.RoleWaliKelas,
		domain.RoleSiswa,
	); err != nil {
		return GetAcademicYearOutput{}, err
	}

	if input.AcademicYearID == uuid.Nil {
		return GetAcademicYearOutput{}, domain.ErrInvalidID
	}

	academicYear, err := s.deps.Repositories.AcademicYears.GetByID(
		ctx,
		input.AcademicYearID,
	)
	if err != nil {
		return GetAcademicYearOutput{}, err
	}

	return GetAcademicYearOutput{
		AcademicYear: &academicYear,
	}, nil
}

func (s *academicYearService) List(
	ctx context.Context,
	input ListAcademicYearsInput,
) (ListAcademicYearsOutput, error) {
	if err := RequireRole(
		input.Actor,
		domain.RoleAdmin,
		domain.RoleWaliKelas,
		domain.RoleSiswa,
	); err != nil {
		return ListAcademicYearsOutput{}, err
	}

	academicYears, err := s.deps.Repositories.AcademicYears.List(
		ctx,
		repository.ListOptions{
			Limit:  input.Options.Limit,
			Offset: input.Options.Offset,
		},
	)
	if err != nil {
		return ListAcademicYearsOutput{}, err
	}

	result := make([]*domain.AcademicYear, 0, len(academicYears))
	for i := range academicYears {
		result = append(result, &academicYears[i])
	}

	return ListAcademicYearsOutput{
		AcademicYears: result,
	}, nil
}

func (s *academicYearService) StartClosing(
	ctx context.Context,
	input StartClosingAcademicYearInput,
) (StartClosingAcademicYearOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return StartClosingAcademicYearOutput{}, err
	}

	if input.AcademicYearID == uuid.Nil {
		return StartClosingAcademicYearOutput{}, domain.ErrInvalidID
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return StartClosingAcademicYearOutput{}, fmt.Errorf(
			"begin start closing academic year transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()

	academicYear, err := repos.AcademicYears.GetByID(
		ctx,
		input.AcademicYearID,
	)
	if err != nil {
		return StartClosingAcademicYearOutput{}, err
	}

	if err := academicYear.StartClosing(); err != nil {
		return StartClosingAcademicYearOutput{}, err
	}

	if err := repos.AcademicYears.Update(ctx, academicYear); err != nil {
		return StartClosingAcademicYearOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return StartClosingAcademicYearOutput{}, fmt.Errorf(
			"commit start closing academic year transaction: %w",
			err,
		)
	}

	committed = true

	return StartClosingAcademicYearOutput{
		AcademicYear: &academicYear,
	}, nil
}

func (s *academicYearService) Close(
	ctx context.Context,
	input CloseAcademicYearInput,
) (CloseAcademicYearOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return CloseAcademicYearOutput{}, err
	}

	if input.AcademicYearID == uuid.Nil {
		return CloseAcademicYearOutput{}, domain.ErrInvalidID
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return CloseAcademicYearOutput{}, fmt.Errorf(
			"begin close academic year transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()

	academicYear, err := repos.AcademicYears.GetByID(
		ctx,
		input.AcademicYearID,
	)
	if err != nil {
		return CloseAcademicYearOutput{}, err
	}

	if err := academicYear.Close(); err != nil {
		return CloseAcademicYearOutput{}, err
	}

	if err := repos.AcademicYears.Update(ctx, academicYear); err != nil {
		return CloseAcademicYearOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return CloseAcademicYearOutput{}, fmt.Errorf(
			"commit close academic year transaction: %w",
			err,
		)
	}

	committed = true

	return CloseAcademicYearOutput{
		AcademicYear: &academicYear,
	}, nil
}

func (s *academicYearService) Reopen(
	ctx context.Context,
	input ReopenAcademicYearInput,
) (ReopenAcademicYearOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ReopenAcademicYearOutput{}, err
	}

	if input.AcademicYearID == uuid.Nil {
		return ReopenAcademicYearOutput{}, domain.ErrInvalidID
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return ReopenAcademicYearOutput{}, fmt.Errorf(
			"begin reopen academic year transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()

	academicYear, err := repos.AcademicYears.GetByID(
		ctx,
		input.AcademicYearID,
	)
	if err != nil {
		return ReopenAcademicYearOutput{}, err
	}

	if err := academicYear.Reopen(); err != nil {
		return ReopenAcademicYearOutput{}, err
	}

	if err := repos.AcademicYears.Update(ctx, academicYear); err != nil {
		return ReopenAcademicYearOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return ReopenAcademicYearOutput{}, fmt.Errorf(
			"commit reopen academic year transaction: %w",
			err,
		)
	}

	committed = true

	return ReopenAcademicYearOutput{
		AcademicYear: &academicYear,
	}, nil
}
