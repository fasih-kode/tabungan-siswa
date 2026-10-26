package http

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type settlementPageServiceFake struct {
	service.SettlementService
	settlement        service.GetSettlementOutput
	preview           service.PreviewYearEndOutput
	leavingPreview    service.PreviewYearEndOutput
	leavingSettlement service.SettleStudentLeavingOutput
	settleErr         error
	leavingSettleErr  error
	readiness         service.YearSettlementReadinessOutput
	readinessErr      error
}

func (f settlementPageServiceFake) Get(context.Context, service.GetSettlementInput) (service.GetSettlementOutput, error) {
	return f.settlement, nil
}

func (f settlementPageServiceFake) PreviewYearEnd(context.Context, service.PreviewYearEndInput) (service.PreviewYearEndOutput, error) {
	return f.preview, nil
}

func (f settlementPageServiceFake) PreviewStudentLeaving(context.Context, service.PreviewStudentLeavingInput) (service.PreviewYearEndOutput, error) {
	return f.leavingPreview, nil
}

func (f settlementPageServiceFake) SettleStudentLeaving(_ context.Context, input service.SettleStudentLeavingInput) (service.SettleStudentLeavingOutput, error) {
	return f.leavingSettlement, f.leavingSettleErr
}

func (f settlementPageServiceFake) SettleYearEnd(context.Context, service.SettleYearEndInput) (service.SettleYearEndOutput, error) {
	return service.SettleYearEndOutput{}, f.settleErr
}

func (f settlementPageServiceFake) YearReadiness(context.Context, service.YearSettlementReadinessInput) (service.YearSettlementReadinessOutput, error) {
	return f.readiness, f.readinessErr
}

type settlementPageAcademicYearFake struct {
	service.AcademicYearService
	year  *domain.AcademicYear
	years []*domain.AcademicYear
}

func (f settlementPageAcademicYearFake) Get(context.Context, service.GetAcademicYearInput) (service.GetAcademicYearOutput, error) {
	return service.GetAcademicYearOutput{AcademicYear: f.year}, nil
}

func (f settlementPageAcademicYearFake) List(context.Context, service.ListAcademicYearsInput) (service.ListAcademicYearsOutput, error) {
	return service.ListAcademicYearsOutput{AcademicYears: f.years}, nil
}

func TestSettlementDetailRendersHistoryAndActorNames(t *testing.T) {
	accountID, yearID, studentID := uuid.New(), uuid.New(), uuid.New()
	executedBy, confirmedBy := uuid.New(), uuid.New()
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	settlement := domain.SavingsSettlement{
		ID: uuid.New(), SavingsAccountID: accountID, Type: domain.SettlementRegularYearEnd,
		Amount: 12500, ExecutedBy: executedBy, ExecutedAt: now, Status: domain.SettlementCompleted,
	}
	year := &domain.AcademicYear{ID: yearID, Name: "2025/2026", Status: domain.AcademicYearClosed}
	account := &domain.SavingsAccount{ID: accountID, StudentID: studentID, AcademicYearID: yearID, Status: domain.SavingsAccountSettled}
	student := &domain.Student{ID: studentID, Name: "Siswa Uji"}
	output := service.GetSettlementOutput{
		Settlement: &settlement, History: []domain.SavingsSettlement{settlement}, Account: account, Student: student,
		ActorNames: map[uuid.UUID]string{executedBy: "admin-uji", confirmedBy: "wali-uji"},
		Movements:  []domain.SettlementCashMovement{{ID: uuid.New(), SavingsSettlementID: settlement.ID, Kind: domain.SettlementCashHandover, Amount: 12500, ConfirmedBy: confirmedBy, ConfirmedAt: now}},
	}
	tmpl := template.Must(template.New("settlement").Parse(`{{define "settlement-page"}}{{.StudentName}}|{{.ExecutedBy}}|{{range .History}}{{.ExecutedBy}}/{{.Status}};{{end}}|{{range .Movements}}{{.Actor}};{{end}}{{end}}`))
	handler, err := NewSettlementPageHandler(settlementPageServiceFake{settlement: output}, settlementPageAcademicYearFake{year: year}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	actor := service.Actor{UserID: uuid.New(), Username: "wali-uji", Role: domain.RoleWaliKelas}
	req := requestWithActor(http.MethodGet, "/settlements/"+settlement.ID.String(), actor)
	req.SetPathValue("settlement_id", settlement.ID.String())
	recorder := httptest.NewRecorder()
	handler.Detail(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	for _, want := range []string{"Siswa Uji", "admin-uji", "Selesai", "wali-uji"} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("response %q does not contain %q", recorder.Body.String(), want)
		}
	}
}

func TestSettlementCreateRendersFreshPreviewWhenBalanceChanged(t *testing.T) {
	accountID := uuid.New()
	previewAt := time.Date(2026, 10, 20, 10, 5, 0, 0, time.UTC)
	fake := settlementPageServiceFake{
		settleErr: repository.ErrConflict,
		preview: service.PreviewYearEndOutput{
			Balance: 23000, PreviewedAt: previewAt,
			Student: &domain.Student{Name: "Siswa Uji"},
		},
	}
	tmpl := template.Must(template.New("settlement").Parse(`{{define "settlement-page"}}{{.Error}}|{{.Amount}}|{{.ExpectedBalance}}|{{.PreviewedAt}}|{{.CSRFToken}}{{end}}`))
	handler, err := NewSettlementPageHandler(fake, settlementPageAcademicYearFake{}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	cookie := security.NewDefaultCSRFCookie()
	cookieWriter := httptest.NewRecorder()
	if err := cookie.Set(cookieWriter, "csrf-fixture"); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"csrf_token": {"csrf-fixture"}, "expected_balance": {"20000"}}
	req := httptest.NewRequest(http.MethodPost, "/savings/"+accountID.String()+"/settlements", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookieWriter.Result().Cookies()[0])
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas}))
	req.SetPathValue("account_id", accountID.String())
	recorder := httptest.NewRecorder()
	middleware.NewCSRFMiddleware(cookie).Protect(http.HandlerFunc(handler.Create)).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", recorder.Code, recorder.Body.String())
	}
	for _, want := range []string{"Saldo berubah", "Rp. 23.000", "23000", previewAt.Format(time.RFC3339), "csrf-fixture"} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("response %q does not contain %q", recorder.Body.String(), want)
		}
	}
}

func TestStudentLeavingSettlementPreviewAndConfirmation(t *testing.T) {
	accountID := uuid.New()
	settlementID := uuid.New()
	previewAt := time.Date(2026, 10, 21, 4, 0, 0, 0, time.UTC)
	fake := settlementPageServiceFake{
		leavingPreview:    service.PreviewYearEndOutput{Balance: 25000, PreviewedAt: previewAt, Student: &domain.Student{Name: "Siswa Pindah"}},
		leavingSettlement: service.SettleStudentLeavingOutput{Settlement: &domain.SavingsSettlement{ID: settlementID}},
	}
	tmpl := template.Must(template.New("settlement").Parse(`{{define "settlement-page"}}{{.Heading}}|{{.SettlementType}}|{{.Amount}}|{{.ExpectedBalance}}|{{.CSRFToken}}{{end}}`))
	handler, err := NewSettlementPageHandler(fake, settlementPageAcademicYearFake{}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas}
	cookie := security.NewDefaultCSRFCookie()
	cookieWriter := httptest.NewRecorder()
	if err := cookie.Set(cookieWriter, "csrf-fixture"); err != nil {
		t.Fatal(err)
	}
	get := requestWithActor(http.MethodGet, "/savings/"+accountID.String()+"/leaving-settlement-preview", actor)
	get.SetPathValue("account_id", accountID.String())
	get.AddCookie(cookieWriter.Result().Cookies()[0])
	getRecorder := httptest.NewRecorder()
	middleware.NewCSRFMiddleware(cookie).Protect(http.HandlerFunc(handler.PreviewStudentLeaving)).ServeHTTP(getRecorder, get)
	if getRecorder.Code != http.StatusOK || !strings.Contains(getRecorder.Body.String(), "STUDENT_LEAVING|Rp. 25.000|25000") {
		t.Fatalf("preview status/body = %d/%q", getRecorder.Code, getRecorder.Body.String())
	}

	form := url.Values{"csrf_token": {"csrf-fixture"}, "expected_balance": {"25000"}}
	post := httptest.NewRequest(http.MethodPost, "/savings/"+accountID.String()+"/leaving-settlements", strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(cookieWriter.Result().Cookies()[0])
	post = post.WithContext(service.WithActor(post.Context(), actor))
	post.SetPathValue("account_id", accountID.String())
	postRecorder := httptest.NewRecorder()
	middleware.NewCSRFMiddleware(cookie).Protect(http.HandlerFunc(handler.SettleStudentLeaving)).ServeHTTP(postRecorder, post)
	if postRecorder.Code != http.StatusSeeOther || postRecorder.Header().Get("Location") != "/settlements/"+settlementID.String()+"?notice=settled" {
		t.Fatalf("confirmation status/location = %d/%q; body=%s", postRecorder.Code, postRecorder.Header().Get("Location"), postRecorder.Body.String())
	}
}

func TestSettlementOverviewRendersRoleScopedDataAndUnattributedGroup(t *testing.T) {
	year := &domain.AcademicYear{ID: uuid.New(), Name: "2025/2026", Status: domain.AcademicYearClosing}
	waliID, otherWaliID := uuid.New(), uuid.New()
	classID := uuid.New()
	accountID := uuid.New()
	rows := []repository.SettlementReadinessAccount{
		{Account: domain.SavingsAccount{ID: accountID, Status: domain.SavingsAccountOpen}, Student: domain.Student{Name: "Siswa Wali"}, ClassID: &classID, ClassName: "VII A", WaliID: &waliID, WaliName: "Wali A"},
		{Account: domain.SavingsAccount{ID: uuid.New(), Status: domain.SavingsAccountOpen}, Student: domain.Student{Name: "Tanpa atribusi"}, AttributionIssue: true},
		{Account: domain.SavingsAccount{ID: uuid.New(), Status: domain.SavingsAccountOpen}, Student: domain.Student{Name: "Siswa lain"}, ClassID: &classID, ClassName: "VII A", WaliID: &otherWaliID, WaliName: "Wali B"},
	}
	tmpl := template.Must(template.New("settlement").Parse(`{{define "settlement-page"}}{{.YearName}}|{{.Total}}|{{.ReadinessLabel}}|{{range .Rows}}{{.StudentName}}/{{.Class}}/{{.Wali}}/{{.PreviewURL}};{{end}}|{{range .Groups}}{{.Wali}}/{{.Class}}/{{.Status}};{{end}}|{{.TransferURL}}{{end}}`))
	for _, tc := range []struct {
		name string
		role domain.UserRole
		rows []repository.SettlementReadinessAccount
		want []string
	}{
		{name: "admin sees unresolved groups", role: domain.RoleAdmin, rows: rows, want: []string{"Tanpa atribusi", "Data penanggung jawab ambigu", "/settlement-preview", "Kesiapan tahun ajaran", "Siswa lain", "Wali A/VII A/Menghambat penutupan"}},
		{name: "wali receives scoped rows", role: domain.RoleWaliKelas, rows: rows[:1], want: []string{"Kesiapan cakupan Anda", "Siswa Wali", "Wali A", "Wali A/VII A/Menghambat penutupan"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := settlementPageServiceFake{readiness: service.YearSettlementReadinessOutput{AcademicYear: year, Rows: tc.rows, Total: len(tc.rows)}}
			handler, err := NewSettlementPageHandler(fake, settlementPageAcademicYearFake{years: []*domain.AcademicYear{year}}, tmpl)
			if err != nil {
				t.Fatal(err)
			}
			actor := service.Actor{UserID: uuid.New(), Role: tc.role}
			req := requestWithActor(http.MethodGet, "/savings?view=settlement&academic_year_id="+year.ID.String(), actor)
			recorder := httptest.NewRecorder()
			handler.Overview(recorder, req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
			}
			for _, want := range tc.want {
				if !strings.Contains(recorder.Body.String(), want) {
					t.Errorf("response %q does not contain %q", recorder.Body.String(), want)
				}
			}
			if tc.role == domain.RoleWaliKelas && strings.Contains(recorder.Body.String(), "Siswa lain") {
				t.Fatal("Wali overview exposed another Wali's account")
			}
		})
	}
}

func TestSettlementOverviewRendersSafeReadinessError(t *testing.T) {
	year := &domain.AcademicYear{ID: uuid.New(), Name: "2025/2026", Status: domain.AcademicYearClosing}
	tmpl := template.Must(template.New("settlement").Parse(`{{define "settlement-page"}}{{.Heading}}{{end}}`))
	fake := settlementPageServiceFake{readinessErr: service.ErrForbidden}
	handler, err := NewSettlementPageHandler(fake, settlementPageAcademicYearFake{years: []*domain.AcademicYear{year}}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	req := requestWithActor(http.MethodGet, "/savings?academic_year_id="+year.ID.String(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
	recorder := httptest.NewRecorder()
	handler.Overview(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "SQL") || strings.Contains(recorder.Body.String(), "stack") {
		t.Fatalf("unsafe error detail rendered: %s", recorder.Body.String())
	}
}
