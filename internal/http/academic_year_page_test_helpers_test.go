package http

import (
	"context"
	"html/template"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/service"
)

type academicYearServiceFake struct {
	years          []*domain.AcademicYear
	createErr      error
	getErr         error
	listErr        error
	transitionErr  error
	lastListInput  service.ListAcademicYearsInput
	lastTransition string
}

func (f *academicYearServiceFake) Create(_ context.Context, input service.CreateAcademicYearInput) (service.CreateAcademicYearOutput, error) {
	if f.createErr != nil {
		return service.CreateAcademicYearOutput{}, f.createErr
	}
	year, err := domain.NewAcademicYear(input.Name, input.StartDate, input.EndDate)
	if err != nil {
		return service.CreateAcademicYearOutput{}, err
	}
	return service.CreateAcademicYearOutput{AcademicYear: &year}, nil
}
func (f *academicYearServiceFake) Get(_ context.Context, _ service.GetAcademicYearInput) (service.GetAcademicYearOutput, error) {
	if f.getErr != nil {
		return service.GetAcademicYearOutput{}, f.getErr
	}
	if len(f.years) == 0 {
		return service.GetAcademicYearOutput{}, nil
	}
	return service.GetAcademicYearOutput{AcademicYear: f.years[0]}, nil
}
func (f *academicYearServiceFake) List(_ context.Context, input service.ListAcademicYearsInput) (service.ListAcademicYearsOutput, error) {
	f.lastListInput = input
	if f.listErr != nil {
		return service.ListAcademicYearsOutput{}, f.listErr
	}
	return service.ListAcademicYearsOutput{AcademicYears: f.years}, nil
}
func (f *academicYearServiceFake) StartClosing(context.Context, service.StartClosingAcademicYearInput) (service.StartClosingAcademicYearOutput, error) {
	f.lastTransition = "start-closing"
	if f.transitionErr != nil {
		return service.StartClosingAcademicYearOutput{}, f.transitionErr
	}
	return service.StartClosingAcademicYearOutput{}, nil
}
func (f *academicYearServiceFake) Close(context.Context, service.CloseAcademicYearInput) (service.CloseAcademicYearOutput, error) {
	f.lastTransition = "close"
	if f.transitionErr != nil {
		return service.CloseAcademicYearOutput{}, f.transitionErr
	}
	return service.CloseAcademicYearOutput{}, nil
}
func (f *academicYearServiceFake) Reopen(context.Context, service.ReopenAcademicYearInput) (service.ReopenAcademicYearOutput, error) {
	f.lastTransition = "reopen"
	if f.transitionErr != nil {
		return service.ReopenAcademicYearOutput{}, f.transitionErr
	}
	return service.ReopenAcademicYearOutput{}, nil
}

func newTestAcademicYearPageHandler(t *testing.T) *AcademicYearPageHandler {
	t.Helper()
	templates := template.Must(template.New("academic-years").Parse(`{{define "academic-years"}}academic years{{end}}`))
	handler, err := NewAcademicYearPageHandler(&academicYearServiceFake{}, templates)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
