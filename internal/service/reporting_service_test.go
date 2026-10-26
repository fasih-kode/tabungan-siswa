package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type reportingTestYears struct {
	repository.AcademicYearRepository
	years []domain.AcademicYear
}

func (f *reportingTestYears) List(context.Context, repository.ListOptions) ([]domain.AcademicYear, error) {
	return f.years, nil
}

type reportingTestRepository struct {
	repository.ReportingRepository
	dashboardCalls int
}

func (f *reportingTestRepository) Dashboard(context.Context, uuid.UUID, uuid.UUID, domain.UserRole) (repository.DashboardMetrics, error) {
	f.dashboardCalls++
	return repository.DashboardMetrics{Students: 3}, nil
}
func (f *reportingTestRepository) StudentReport(context.Context, uuid.UUID, uuid.UUID, domain.UserRole) ([]repository.StudentReportRow, error) {
	return nil, nil
}
func (f *reportingTestRepository) ClassReport(context.Context, uuid.UUID, uuid.UUID, domain.UserRole) ([]repository.ClassReportRow, error) {
	return nil, nil
}
func (f *reportingTestRepository) SettlementReport(context.Context, uuid.UUID, uuid.UUID, domain.UserRole) ([]repository.SettlementReportRow, error) {
	return nil, nil
}

func TestReportingServiceIncludesClosingYearAndRejectsStudentDirectoryReport(t *testing.T) {
	year, err := domain.NewAcademicYear("Sedang ditutup", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	year.Status = domain.AcademicYearClosing
	reports := &reportingTestRepository{}
	svc, err := service.NewReportingService(service.Dependencies{Repositories: repository.RepositorySet{AcademicYears: &reportingTestYears{years: []domain.AcademicYear{year}}, Reporting: reports}})
	if err != nil {
		t.Fatal(err)
	}
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	out, err := svc.Dashboard(context.Background(), actor)
	if err != nil {
		t.Fatalf("Dashboard(): %v", err)
	}
	if out.AcademicYear == nil || out.AcademicYear.ID != year.ID || out.Metrics.Students != 3 {
		t.Fatalf("Dashboard() = %+v", out)
	}

	_, err = svc.StudentReport(context.Background(), service.Actor{UserID: uuid.New(), Role: domain.RoleSiswa})
	if !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("StudentReport() for SISWA error = %v, want forbidden", err)
	}
}

func TestReportingServiceShowsEmptyWhenNoActiveYear(t *testing.T) {
	reports := &reportingTestRepository{}
	svc, err := service.NewReportingService(service.Dependencies{Repositories: repository.RepositorySet{AcademicYears: &reportingTestYears{}, Reporting: reports}})
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.Dashboard(context.Background(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if out.AcademicYear != nil || reports.dashboardCalls != 0 {
		t.Fatalf("unexpected no-year output/calls: %+v/%d", out, reports.dashboardCalls)
	}
}
