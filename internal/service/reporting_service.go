package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
)

type reportingService struct{ deps Dependencies }

var _ ReportingService = (*reportingService)(nil)

func NewReportingService(deps Dependencies) (*reportingService, error) {
	if deps.Repositories.Reporting == nil || deps.Repositories.AcademicYears == nil {
		return nil, ErrInvalidDependency
	}
	return &reportingService{deps: deps}, nil
}

func (s *reportingService) activeYear(ctx context.Context, actor Actor) (*domain.AcademicYear, error) {
	if err := RequireRole(actor, domain.RoleAdmin, domain.RoleWaliKelas, domain.RoleSiswa); err != nil {
		return nil, err
	}
	years, err := s.deps.Repositories.AcademicYears.List(ctx, repository.ListOptions{Limit: 250})
	if err != nil {
		return nil, fmt.Errorf("list academic years for reporting: %w", err)
	}
	var active *domain.AcademicYear
	for i := range years {
		if years[i].Status == domain.AcademicYearClosed {
			continue
		}
		if active != nil {
			return nil, fmt.Errorf("reporting requires at most one non-closed academic year")
		}
		year := years[i]
		active = &year
	}
	return active, nil
}

func (s *reportingService) Dashboard(ctx context.Context, actor Actor) (DashboardOutput, error) {
	year, err := s.activeYear(ctx, actor)
	if err != nil || year == nil {
		return DashboardOutput{AcademicYear: year}, err
	}
	metrics, err := s.deps.Repositories.Reporting.Dashboard(ctx, year.ID, actor.UserID, actor.Role)
	if err != nil {
		return DashboardOutput{}, err
	}
	out := DashboardOutput{AcademicYear: year, Metrics: metrics}
	if actor.Role == domain.RoleSiswa {
		student, err := s.deps.Repositories.Students.GetByUserID(ctx, actor.UserID)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return DashboardOutput{}, err
		}
		if err == nil {
			account, err := s.deps.Repositories.SavingsAccounts.GetByStudentAndAcademicYear(ctx, student.ID, year.ID)
			if err != nil && !errors.Is(err, repository.ErrNotFound) {
				return DashboardOutput{}, err
			}
			if err == nil {
				transactions, err := s.deps.Repositories.Transactions.ListBySavingsAccount(ctx, account.ID, repository.ListOptions{Limit: 5})
				if err != nil {
					return DashboardOutput{}, err
				}
				out.Recent = transactions
			}
		}
	}
	return out, nil
}

func (s *reportingService) StudentReport(ctx context.Context, actor Actor) (StudentReportOutput, error) {
	if err := RequireRole(actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		return StudentReportOutput{}, err
	}
	year, err := s.activeYear(ctx, actor)
	if err != nil || year == nil {
		return StudentReportOutput{AcademicYear: year}, err
	}
	rows, err := s.deps.Repositories.Reporting.StudentReport(ctx, year.ID, actor.UserID, actor.Role)
	if err != nil {
		return StudentReportOutput{}, err
	}
	return StudentReportOutput{AcademicYear: year, Rows: rows}, nil
}

func (s *reportingService) ClassReport(ctx context.Context, actor Actor) (ClassReportOutput, error) {
	if err := RequireRole(actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		return ClassReportOutput{}, err
	}
	year, err := s.activeYear(ctx, actor)
	if err != nil || year == nil {
		return ClassReportOutput{AcademicYear: year}, err
	}
	rows, err := s.deps.Repositories.Reporting.ClassReport(ctx, year.ID, actor.UserID, actor.Role)
	if err != nil {
		return ClassReportOutput{}, err
	}
	return ClassReportOutput{AcademicYear: year, Rows: rows}, nil
}

func (s *reportingService) AcademicYearReport(ctx context.Context, actor Actor) (DashboardOutput, error) {
	if err := RequireRole(actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		return DashboardOutput{}, err
	}
	return s.Dashboard(ctx, actor)
}

func (s *reportingService) SettlementReport(ctx context.Context, actor Actor) (SettlementReportOutput, error) {
	if err := RequireRole(actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		return SettlementReportOutput{}, err
	}
	year, err := s.activeYear(ctx, actor)
	if err != nil || year == nil {
		return SettlementReportOutput{AcademicYear: year}, err
	}
	rows, err := s.deps.Repositories.Reporting.SettlementReport(ctx, year.ID, actor.UserID, actor.Role)
	if err != nil {
		return SettlementReportOutput{}, err
	}
	return SettlementReportOutput{AcademicYear: year, Rows: rows}, nil
}
