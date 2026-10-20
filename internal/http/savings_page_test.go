package http

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type savingsPageAccountsFake struct {
	service.SavingsAccountService
	account        domain.SavingsAccount
	createInput    service.CreateSavingsAccountInput
	getInput       service.GetSavingsAccountInput
	passbookInput  service.GetSavingsAccountPassbookInput
	missingStudent uuid.UUID
	createCalls    int
}

func (f *savingsPageAccountsFake) Create(_ context.Context, input service.CreateSavingsAccountInput) (service.CreateSavingsAccountOutput, error) {
	f.createCalls++
	f.createInput = input
	return service.CreateSavingsAccountOutput{Account: &f.account}, nil
}
func (f *savingsPageAccountsFake) Get(_ context.Context, input service.GetSavingsAccountInput) (service.GetSavingsAccountOutput, error) {
	f.getInput = input
	return service.GetSavingsAccountOutput{Account: &f.account}, nil
}
func (f *savingsPageAccountsFake) GetByStudentAndAcademicYear(_ context.Context, input service.GetSavingsAccountByStudentAndAcademicYearInput) (service.GetSavingsAccountByStudentAndAcademicYearOutput, error) {
	if input.StudentID == f.missingStudent {
		return service.GetSavingsAccountByStudentAndAcademicYearOutput{}, repository.ErrNotFound
	}
	return service.GetSavingsAccountByStudentAndAcademicYearOutput{Account: &f.account}, nil
}
func (f *savingsPageAccountsFake) GetBalance(context.Context, service.GetSavingsAccountBalanceInput) (service.GetSavingsAccountBalanceOutput, error) {
	return service.GetSavingsAccountBalanceOutput{Balance: 1250000}, nil
}
func (f *savingsPageAccountsFake) GetPassbook(_ context.Context, input service.GetSavingsAccountPassbookInput) (service.GetSavingsAccountPassbookOutput, error) {
	f.passbookInput = input
	tx := domain.Transaction{ID: uuid.New(), Type: domain.TransactionDeposit, Amount: 1250000}
	return service.GetSavingsAccountPassbookOutput{Balance: 1250000, Entries: []*service.SavingsPassbookEntry{{Transaction: &tx, RunningBalance: 1250000}}}, nil
}

type savingsPageStudentsFake struct {
	service.StudentService
	listInput  service.ListStudentsInput
	students   []*domain.Student
	placements map[uuid.UUID]service.StudentPlacement
	self       *domain.Student
	getStudent *domain.Student
}

func (f *savingsPageStudentsFake) List(_ context.Context, input service.ListStudentsInput) (service.ListStudentsOutput, error) {
	f.listInput = input
	return service.ListStudentsOutput{Students: f.students, Placements: f.placements, Total: len(f.students)}, nil
}
func (f *savingsPageStudentsFake) GetByUserID(context.Context, service.GetStudentByUserIDInput) (service.GetStudentByUserIDOutput, error) {
	return service.GetStudentByUserIDOutput{Student: f.self}, nil
}
func (f *savingsPageStudentsFake) Get(context.Context, service.GetStudentInput) (service.GetStudentOutput, error) {
	return service.GetStudentOutput{Student: f.getStudent}, nil
}

type savingsPageYearsFake struct {
	service.AcademicYearService
	years []*domain.AcademicYear
	get   *domain.AcademicYear
}

func (f *savingsPageYearsFake) List(context.Context, service.ListAcademicYearsInput) (service.ListAcademicYearsOutput, error) {
	return service.ListAcademicYearsOutput{AcademicYears: f.years}, nil
}
func (f *savingsPageYearsFake) Get(context.Context, service.GetAcademicYearInput) (service.GetAcademicYearOutput, error) {
	return service.GetAcademicYearOutput{AcademicYear: f.get}, nil
}

type savingsPageClassesFake struct{ service.ClassService }

func (*savingsPageClassesFake) List(context.Context, service.ListClassesInput) (service.ListClassesOutput, error) {
	return service.ListClassesOutput{}, nil
}

func newSavingsPageHandlerForTest(t *testing.T, accounts service.SavingsAccountService, students service.StudentService, years service.AcademicYearService) *SavingsPageHandler {
	t.Helper()
	tmpl := template.Must(template.New("savings").Parse(`{{define "savings"}}{{.Heading}}|{{.AcademicYearID}}|{{.StudentName}}|{{.Balance}}{{range .Rows}}|{{.Name}}:{{.AccountID}}:{{.Balance}}:{{.CanCreate}}{{end}}{{range .Passbook}}|{{.Type}}:{{.Amount}}:{{.Balance}}{{end}}{{end}}`))
	h, err := NewSavingsPageHandler(accounts, students, years, &savingsPageClassesFake{}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestSavingsPageListScopesRosterAndShowsAccountOrCreateAction(t *testing.T) {
	yearID, studentID, missingID := uuid.New(), uuid.New(), uuid.New()
	year := &domain.AcademicYear{ID: yearID, Name: "2026/2027", Status: domain.AcademicYearOpen}
	account := domain.SavingsAccount{ID: uuid.New(), StudentID: studentID, AcademicYearID: yearID, Status: domain.SavingsAccountOpen}
	accounts := &savingsPageAccountsFake{account: account, missingStudent: missingID}
	students := &savingsPageStudentsFake{students: []*domain.Student{{ID: studentID, Name: "Ani", Status: domain.StudentActive}, {ID: missingID, Name: "Budi", Status: domain.StudentActive}}, placements: map[uuid.UUID]service.StudentPlacement{studentID: {Class: "VII A"}, missingID: {Class: "VII A"}}}
	h := newSavingsPageHandlerForTest(t, accounts, students, &savingsPageYearsFake{years: []*domain.AcademicYear{year}, get: year})
	req := requestWithActor(http.MethodGet, "/savings", service.Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas})
	recorder := httptest.NewRecorder()
	withCSRFForTest(http.HandlerFunc(h.List)).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("List status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if students.listInput.Actor.Role != domain.RoleWaliKelas || !students.listInput.IncludePlacement || students.listInput.AcademicYearID != yearID {
		t.Fatalf("student query was not scoped by year/role: %+v", students.listInput)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Ani:") || !strings.Contains(body, "Rp 1.250.000") || !strings.Contains(body, "Budi::") || !strings.Contains(body, ":true") {
		t.Fatalf("list did not render account and eligible create state: %s", body)
	}
}

func TestSavingsPageCreateAndDetail(t *testing.T) {
	studentID, yearID := uuid.New(), uuid.New()
	year := &domain.AcademicYear{ID: yearID, Name: "2026/2027", Status: domain.AcademicYearOpen}
	student := &domain.Student{ID: studentID, Name: "Ani"}
	account := domain.SavingsAccount{ID: uuid.New(), StudentID: studentID, AcademicYearID: yearID, Status: domain.SavingsAccountOpen}
	accounts := &savingsPageAccountsFake{account: account}
	h := newSavingsPageHandlerForTest(t, accounts, &savingsPageStudentsFake{getStudent: student}, &savingsPageYearsFake{get: year})
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas}
	form := url.Values{"student_id": {studentID.String()}, "academic_year_id": {yearID.String()}}
	req := httptest.NewRequest(http.MethodPost, "/savings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), actor))
	recorder := httptest.NewRecorder()
	h.Create(recorder, req)
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/savings?academic_year_id="+yearID.String()+"&notice=created" || accounts.createCalls != 1 || accounts.createInput.Actor != actor {
		t.Fatalf("Create status/location/calls=%d/%q/%d", recorder.Code, recorder.Header().Get("Location"), accounts.createCalls)
	}

	detailReq := requestWithActor(http.MethodGet, "/savings/"+account.ID.String(), actor)
	detailReq.SetPathValue("id", account.ID.String())
	detailRecorder := httptest.NewRecorder()
	withCSRFForTest(http.HandlerFunc(h.Detail)).ServeHTTP(detailRecorder, detailReq)
	if detailRecorder.Code != http.StatusOK || accounts.getInput.SavingsAccountID != account.ID || accounts.passbookInput.SavingsAccountID != account.ID || !strings.Contains(detailRecorder.Body.String(), "Ani") || !strings.Contains(detailRecorder.Body.String(), "Setor:Rp 1.250.000:Rp 1.250.000") {
		t.Fatalf("Detail status/body=%d/%s", detailRecorder.Code, detailRecorder.Body.String())
	}
}

func TestSavingsPageRejectsMalformedAccountCreate(t *testing.T) {
	h := newSavingsPageHandlerForTest(t, &savingsPageAccountsFake{}, &savingsPageStudentsFake{}, &savingsPageYearsFake{})
	form := url.Values{"student_id": {"bad"}, "academic_year_id": {uuid.NewString()}}
	req := httptest.NewRequest(http.MethodPost, "/savings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas}))
	recorder := httptest.NewRecorder()
	h.Create(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Create malformed status=%d", recorder.Code)
	}
}
