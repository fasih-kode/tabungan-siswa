package http

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type classTransferPageServiceFake struct {
	listStudentInput    service.ListStudentTransfersInput
	listPendingInput    service.ListPendingTransfersInput
	getInput            service.GetTransferInput
	requestInput        service.RequestTransferInput
	requestOptionsInput service.RequestTransferOptionsInput
	approveInput        service.ApproveTransferInput
	rejectInput         service.RejectTransferInput
	listStudentOut      service.ListStudentTransfersOutput
	listPendingOut      service.ListPendingTransfersOutput
	requestOut          service.RequestTransferOutput
	requestOptionsOut   service.RequestTransferOptionsOutput
	getOut              service.GetTransferOutput
	approveOut          service.ApproveTransferOutput
	rejectOut           service.RejectTransferOutput
	listStudentErr      error
	listPendingErr      error
	requestErr          error
	requestOptionsErr   error
	getErr              error
	approveErr          error
	rejectErr           error
	listStudentCalls    int
	listPendingCalls    int
	requestCalls        int
	getCalls            int
	approveCalls        int
	rejectCalls         int
}

func (f *classTransferPageServiceFake) Request(_ context.Context, input service.RequestTransferInput) (service.RequestTransferOutput, error) {
	f.requestCalls++
	f.requestInput = input
	return f.requestOut, f.requestErr
}
func (f *classTransferPageServiceFake) RequestOptions(_ context.Context, input service.RequestTransferOptionsInput) (service.RequestTransferOptionsOutput, error) {
	f.requestOptionsInput = input
	return f.requestOptionsOut, f.requestOptionsErr
}
func (f *classTransferPageServiceFake) Get(_ context.Context, input service.GetTransferInput) (service.GetTransferOutput, error) {
	f.getCalls++
	f.getInput = input
	return f.getOut, f.getErr
}
func (f *classTransferPageServiceFake) ListByStudent(_ context.Context, input service.ListStudentTransfersInput) (service.ListStudentTransfersOutput, error) {
	f.listStudentCalls++
	f.listStudentInput = input
	return f.listStudentOut, f.listStudentErr
}
func (f *classTransferPageServiceFake) ListPending(_ context.Context, input service.ListPendingTransfersInput) (service.ListPendingTransfersOutput, error) {
	f.listPendingCalls++
	f.listPendingInput = input
	return f.listPendingOut, f.listPendingErr
}
func (f *classTransferPageServiceFake) Approve(_ context.Context, input service.ApproveTransferInput) (service.ApproveTransferOutput, error) {
	f.approveCalls++
	f.approveInput = input
	return f.approveOut, f.approveErr
}
func (f *classTransferPageServiceFake) Reject(_ context.Context, input service.RejectTransferInput) (service.RejectTransferOutput, error) {
	f.rejectCalls++
	f.rejectInput = input
	return f.rejectOut, f.rejectErr
}

func newClassTransferPageHandlerForTest(t *testing.T, fake *classTransferPageServiceFake) *ClassTransferPageHandler {
	t.Helper()
	tmpl := template.Must(template.New("class-transfers").Parse(`{{define "class-transfers"}}{{.Heading}}|{{.Role}}|{{.StudentID}}|{{.AcademicYearID}}|{{.Page}}|{{.Limit}}|{{if .HasNext}}next{{end}}{{range .Rows}}|{{.ID}}:{{.CanReview}}{{end}}{{end}}`))
	handler, err := NewClassTransferPageHandler(fake, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func withCSRFForTest(handler http.Handler) http.Handler {
	return middleware.NewCSRFMiddleware(security.NewDefaultCSRFCookie()).Protect(handler)
}

func TestClassTransferPageListStudentPassesPaginationAndRenders(t *testing.T) {
	studentID, yearID := uuid.New(), uuid.New()
	fake := &classTransferPageServiceFake{listStudentOut: service.ListStudentTransfersOutput{TransferRequests: []*domain.ClassTransferRequest{
		{ID: uuid.New(), StudentID: studentID, AcademicYearID: yearID, Status: domain.TransferPending, CreatedAt: time.Now()},
		{ID: uuid.New(), StudentID: studentID, AcademicYearID: yearID, Status: domain.TransferPending, CreatedAt: time.Now()},
		{ID: uuid.New(), StudentID: studentID, AcademicYearID: yearID, Status: domain.TransferPending, CreatedAt: time.Now()},
	}}}
	for len(fake.listStudentOut.TransferRequests) < 11 {
		fake.listStudentOut.TransferRequests = append(fake.listStudentOut.TransferRequests, &domain.ClassTransferRequest{ID: uuid.New(), StudentID: studentID, AcademicYearID: yearID, Status: domain.TransferPending})
	}
	fake.listStudentOut.Total = 21
	handler := newClassTransferPageHandlerForTest(t, fake)
	actor := service.Actor{UserID: uuid.New(), Username: "wali", Role: domain.RoleWaliKelas}
	req := requestWithActor(http.MethodGet, "/students/"+studentID.String()+"/class-transfer-requests?academic_year_id="+yearID.String()+"&page=2&limit=2", actor)
	req.SetPathValue("student_id", studentID.String())
	recorder := httptest.NewRecorder()
	withCSRFForTest(http.HandlerFunc(handler.ListStudent)).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("ListStudent() status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if fake.listStudentInput.Actor != actor || fake.listStudentInput.StudentID != studentID || fake.listStudentInput.AcademicYearID != yearID || fake.listStudentInput.Options.Limit != 11 || fake.listStudentInput.Options.Offset != 10 {
		t.Fatalf("ListByStudent input = %+v, want actor/student/year and lookahead limit/offset", fake.listStudentInput)
	}
	if !strings.Contains(recorder.Body.String(), "Transfer Kelas Siswa|WALI_KELAS|") || !strings.Contains(recorder.Body.String(), "next") || strings.Count(recorder.Body.String(), ":false") != 10 {
		t.Fatalf("rendered list missing expected view model state: %s", recorder.Body.String())
	}
}

func TestClassTransferPageListPendingRequiresAdminAndUsesYear(t *testing.T) {
	yearID := uuid.New()
	fake := &classTransferPageServiceFake{listPendingOut: service.ListPendingTransfersOutput{Total: 30}}
	handler := newClassTransferPageHandlerForTest(t, fake)

	waliReq := requestWithActor(http.MethodGet, "/class-transfer-requests/pending?academic_year_id="+yearID.String(), service.Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas})
	waliRecorder := httptest.NewRecorder()
	handler.ListPending(waliRecorder, waliReq)
	if waliRecorder.Code != http.StatusForbidden || fake.listPendingCalls != 0 {
		t.Fatalf("Wali pending list status/calls = %d/%d, want 403/0", waliRecorder.Code, fake.listPendingCalls)
	}

	admin := service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	adminReq := requestWithActor(http.MethodGet, "/class-transfer-requests/pending?academic_year_id="+yearID.String()+"&page=3&limit=5", admin)
	adminRecorder := httptest.NewRecorder()
	withCSRFForTest(http.HandlerFunc(handler.ListPending)).ServeHTTP(adminRecorder, adminReq)
	if adminRecorder.Code != http.StatusOK {
		t.Fatalf("Admin pending list status = %d, want 200", adminRecorder.Code)
	}
	if fake.listPendingCalls != 1 || fake.listPendingInput.Actor != admin || fake.listPendingInput.AcademicYearID != yearID || fake.listPendingInput.Options.Limit != 11 || fake.listPendingInput.Options.Offset != 20 {
		t.Fatalf("ListPending input/calls = %+v/%d, want year, actor, lookahead, and offset", fake.listPendingInput, fake.listPendingCalls)
	}
}

func TestClassTransferPageRequestParsesFormAndRedirects(t *testing.T) {
	requestID := uuid.New()
	fake := &classTransferPageServiceFake{requestOut: service.RequestTransferOutput{TransferRequest: &domain.ClassTransferRequest{ID: requestID}}}
	handler := newClassTransferPageHandlerForTest(t, fake)
	studentID, yearID, fromID, toID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	form := url.Values{
		"student_id": {studentID.String()}, "academic_year_id": {yearID.String()},
		"from_class_id": {fromID.String()}, "to_class_id": {toID.String()},
	}
	req := httptest.NewRequest(http.MethodPost, "/class-transfer-requests", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	actor := service.Actor{UserID: uuid.New(), Username: "wali", Role: domain.RoleWaliKelas}
	req = req.WithContext(service.WithActor(req.Context(), actor))
	recorder := httptest.NewRecorder()
	handler.Request(recorder, req)
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/class-transfer-requests/"+requestID.String() {
		t.Fatalf("Wali Request() status/location = %d/%q, want 303/detail", recorder.Code, recorder.Header().Get("Location"))
	}
	if fake.requestCalls != 1 || fake.requestInput.Actor != actor || fake.requestInput.StudentID != studentID || fake.requestInput.AcademicYearID != yearID || fake.requestInput.FromClassID != fromID || fake.requestInput.ToClassID != toID {
		t.Fatalf("Request() input/calls = %+v/%d, want parsed values and authenticated actor", fake.requestInput, fake.requestCalls)
	}

	adminReq := httptest.NewRequest(http.MethodPost, "/class-transfer-requests", strings.NewReader(form.Encode()))
	adminReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	adminReq = adminReq.WithContext(service.WithActor(adminReq.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	adminRecorder := httptest.NewRecorder()
	handler.Request(adminRecorder, adminReq)
	if adminRecorder.Code != http.StatusForbidden || fake.requestCalls != 1 {
		t.Fatalf("Admin Request() status/calls = %d/%d, want 403/no additional call", adminRecorder.Code, fake.requestCalls)
	}
}

func TestClassTransferPageApproveRejectParseInputsAndRedirect(t *testing.T) {
	requestID := uuid.New()
	fake := &classTransferPageServiceFake{}
	handler := newClassTransferPageHandlerForTest(t, fake)
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}

	approveForm := url.Values{"effective_date": {"2027-02-03"}}
	approveReq := httptest.NewRequest(http.MethodPost, "/class-transfer-requests/"+requestID.String()+"/approve", strings.NewReader(approveForm.Encode()))
	approveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approveReq.SetPathValue("id", requestID.String())
	approveReq = approveReq.WithContext(service.WithActor(approveReq.Context(), actor))
	approveRecorder := httptest.NewRecorder()
	handler.Approve(approveRecorder, approveReq)
	if approveRecorder.Code != http.StatusSeeOther || approveRecorder.Header().Get("Location") != "/class-transfer-requests/"+requestID.String() {
		t.Fatalf("Approve() status/location = %d/%q, want 303/detail", approveRecorder.Code, approveRecorder.Header().Get("Location"))
	}
	if fake.approveCalls != 1 || fake.approveInput.Actor != actor || fake.approveInput.TransferRequestID != requestID || fake.approveInput.EffectiveDate.Format("2006-01-02") != "2027-02-03" {
		t.Fatalf("Approve() input/calls = %+v/%d, want parsed date and actor", fake.approveInput, fake.approveCalls)
	}

	reason := "Jadwal tidak sesuai"
	rejectForm := url.Values{"rejection_reason": {reason}}
	rejectReq := httptest.NewRequest(http.MethodPost, "/class-transfer-requests/"+requestID.String()+"/reject", strings.NewReader(rejectForm.Encode()))
	rejectReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rejectReq.SetPathValue("id", requestID.String())
	rejectReq = rejectReq.WithContext(service.WithActor(rejectReq.Context(), actor))
	rejectRecorder := httptest.NewRecorder()
	handler.Reject(rejectRecorder, rejectReq)
	if rejectRecorder.Code != http.StatusSeeOther || rejectRecorder.Header().Get("Location") != "/class-transfer-requests/"+requestID.String() {
		t.Fatalf("Reject() status/location = %d/%q, want 303/detail", rejectRecorder.Code, rejectRecorder.Header().Get("Location"))
	}
	if fake.rejectCalls != 1 || fake.rejectInput.Actor != actor || fake.rejectInput.TransferRequestID != requestID || fake.rejectInput.RejectionReason != reason {
		t.Fatalf("Reject() input/calls = %+v/%d, want parsed reason and actor", fake.rejectInput, fake.rejectCalls)
	}
}

func TestClassTransferPageRejectsInvalidInputWithoutCallingService(t *testing.T) {
	fake := &classTransferPageServiceFake{}
	handler := newClassTransferPageHandlerForTest(t, fake)
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}

	req := requestWithActor(http.MethodGet, "/students/not-a-uuid/class-transfer-requests?academic_year_id="+uuid.NewString(), actor)
	recorder := httptest.NewRecorder()
	handler.ListStudent(recorder, req)
	if recorder.Code != http.StatusBadRequest || fake.listStudentCalls != 0 {
		t.Fatalf("invalid list status/calls = %d/%d, want 400/0", recorder.Code, fake.listStudentCalls)
	}

	badDateForm := url.Values{"effective_date": {"not-a-date"}}
	badDateReq := httptest.NewRequest(http.MethodPost, "/class-transfer-requests/"+uuid.NewString()+"/approve", strings.NewReader(badDateForm.Encode()))
	badDateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badDateReq.SetPathValue("id", uuid.NewString())
	badDateReq = badDateReq.WithContext(service.WithActor(badDateReq.Context(), actor))
	badDateRecorder := httptest.NewRecorder()
	handler.Approve(badDateRecorder, badDateReq)
	if badDateRecorder.Code != http.StatusBadRequest || fake.approveCalls != 0 {
		t.Fatalf("invalid date status/calls = %d/%d, want 400/0", badDateRecorder.Code, fake.approveCalls)
	}

	fake.requestErr = domain.ErrInvalidTransition
	requestForm := url.Values{"student_id": {uuid.NewString()}, "academic_year_id": {uuid.NewString()}, "from_class_id": {uuid.NewString()}, "to_class_id": {uuid.NewString()}}
	postReq := httptest.NewRequest(http.MethodPost, "/class-transfer-requests", strings.NewReader(requestForm.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq = postReq.WithContext(service.WithActor(postReq.Context(), actor))
	postRecorder := httptest.NewRecorder()
	handler.Request(postRecorder, postReq)
	if postRecorder.Code < 400 || strings.Contains(postRecorder.Body.String(), "invalid transition") {
		t.Fatalf("service error response status/body = %d/%q, want safe error", postRecorder.Code, postRecorder.Body.String())
	}

	fake.rejectErr = errors.New("sensitive repository details")
	rejectForm := url.Values{"rejection_reason": {"reason"}}
	rejectReq := httptest.NewRequest(http.MethodPost, "/class-transfer-requests/"+uuid.NewString()+"/reject", strings.NewReader(rejectForm.Encode()))
	rejectReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rejectReq.SetPathValue("id", uuid.NewString())
	rejectReq = rejectReq.WithContext(service.WithActor(rejectReq.Context(), actor))
	rejectRecorder := httptest.NewRecorder()
	handler.Reject(rejectRecorder, rejectReq)
	if rejectRecorder.Code != http.StatusInternalServerError || strings.Contains(rejectRecorder.Body.String(), "sensitive repository details") {
		t.Fatalf("unknown error response status/body = %d/%q, want generic 500", rejectRecorder.Code, rejectRecorder.Body.String())
	}
}

func newClassTransferRouterWithSession(t *testing.T, role domain.UserRole, fake *classTransferPageServiceFake, savings ...*SavingsPageHandler) (http.Handler, string) {
	t.Helper()
	userID := uuid.New()
	sessionToken := "class-transfer-session-" + uuid.NewString()
	session := domain.Session{ID: uuid.New(), UserID: userID, TokenHash: security.HashSessionToken(sessionToken), ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	user := domain.User{ID: userID, Username: "class-transfer-test-user", Role: role}
	auth, err := middleware.NewAuthenticationMiddleware(security.NewDefaultSessionCookie(), &routerSessionRepositoryFake{session: session}, &routerUserRepositoryFake{user: user})
	if err != nil {
		t.Fatal(err)
	}
	templates, err := template.New("pages").Parse(
		`{{define "auth"}}auth{{end}}{{define "base"}}base{{end}}{{define "academic-years"}}academic years{{end}}{{define "class-transfers"}}{{.Heading}}|{{.Role}}{{end}}{{define "savings"}}{{.Heading}}{{end}}`,
	)
	if err != nil {
		t.Fatal(err)
	}
	classTransfers, err := NewClassTransferPageHandler(fake, templates)
	if err != nil {
		t.Fatal(err)
	}
	academicYears, err := NewAcademicYearPageHandler(&academicYearServiceFake{}, templates)
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err := NewDashboardPageHandler(templates)
	if err != nil {
		t.Fatal(err)
	}
	handlers := HandlerSet{
		Authentication: &AuthenticationHandler{}, LoginPage: newTestLoginPageHandler(t), Dashboard: dashboard,
		AcademicYear: academicYears, Class: newTestClassPageHandler(t), ClassTransfer: classTransfers,
	}
	if len(savings) > 0 {
		handlers.Savings = savings[0]
	}
	router, err := NewRouter(handlers, MiddlewareSet{Authentication: auth, CSRF: middleware.NewCSRFMiddleware(security.NewDefaultCSRFCookie())}, newTestStaticAssetHandler())
	if err != nil {
		t.Fatal(err)
	}
	return router, sessionToken
}

func TestSavingsRoutesRequireAuthenticationAndCSRF(t *testing.T) {
	yearID, studentID := uuid.New(), uuid.New()
	accountFake := &savingsPageAccountsFake{account: domain.SavingsAccount{ID: uuid.New(), StudentID: studentID, AcademicYearID: yearID}}
	year := &domain.AcademicYear{ID: yearID, Name: "2026/2027", Status: domain.AcademicYearOpen}
	savings := newSavingsPageHandlerForTest(t, accountFake, &savingsPageStudentsFake{}, &savingsPageYearsFake{years: []*domain.AcademicYear{year}, get: year})
	router, session := newClassTransferRouterWithSession(t, domain.RoleWaliKelas, &classTransferPageServiceFake{}, savings)
	unauthenticated := httptest.NewRecorder()
	router.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/savings", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET status=%d", unauthenticated.Code)
	}
	get := httptest.NewRequest(http.MethodGet, "/savings", nil)
	get.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: session})
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, get)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("authenticated GET status=%d body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	form := url.Values{"student_id": {studentID.String()}, "academic_year_id": {yearID.String()}}
	missing := httptest.NewRequest(http.MethodPost, "/savings", strings.NewReader(form.Encode()))
	missing.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	missing.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: session})
	missingRecorder := httptest.NewRecorder()
	router.ServeHTTP(missingRecorder, missing)
	if missingRecorder.Code != http.StatusForbidden || accountFake.createCalls != 0 {
		t.Fatalf("POST without CSRF status/calls=%d/%d", missingRecorder.Code, accountFake.createCalls)
	}
	token, err := security.GenerateCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	form.Set("csrf_token", token)
	valid := httptest.NewRequest(http.MethodPost, "/savings", strings.NewReader(form.Encode()))
	valid.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	valid.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: session})
	valid.AddCookie(&http.Cookie{Name: security.DefaultCSRFCookieName, Value: token})
	validRecorder := httptest.NewRecorder()
	router.ServeHTTP(validRecorder, valid)
	if validRecorder.Code != http.StatusSeeOther || accountFake.createCalls != 1 || accountFake.createInput.Actor.Role != domain.RoleWaliKelas {
		t.Fatalf("POST with CSRF status/calls/role=%d/%d/%s", validRecorder.Code, accountFake.createCalls, accountFake.createInput.Actor.Role)
	}
}

func TestSavingsRouteValidatesFormBeforeCSRF(t *testing.T) {
	accounts := &savingsPageAccountsFake{}
	savings := newSavingsPageHandlerForTest(t, accounts, &savingsPageStudentsFake{}, &savingsPageYearsFake{})
	router, session := newClassTransferRouterWithSession(t, domain.RoleWaliKelas, &classTransferPageServiceFake{}, savings)
	token, err := security.GenerateCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"student_id": {uuid.NewString()}, "academic_year_id": {uuid.NewString()}, "csrf_token": {token}}
	var multipartBody bytes.Buffer
	writer := multipart.NewWriter(&multipartBody)
	for key := range form {
		if err := writer.WriteField(key, form.Get(key)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, contentType, body string
		want                    int
	}{
		{"missing content type", "", form.Encode(), http.StatusBadRequest},
		{"json", "application/json", `{}`, http.StatusBadRequest},
		{"multipart with valid fields and CSRF", writer.FormDataContentType(), multipartBody.String(), http.StatusBadRequest},
		{"invalid media type parameters", "application/x-www-form-urlencoded; charset", form.Encode(), http.StatusBadRequest},
		{"malformed encoding", "application/x-www-form-urlencoded", form.Encode() + "&bad=%zz", http.StatusBadRequest},
		{"oversized after valid fields", "application/x-www-form-urlencoded", form.Encode() + "&padding=" + strings.Repeat("x", maxSavingsFormBytes), http.StatusBadRequest},
		{"valid charset parameter", "application/x-www-form-urlencoded; charset=UTF-8", form.Encode(), http.StatusSeeOther},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := accounts.createCalls
			req := httptest.NewRequest(http.MethodPost, "/savings", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			req.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: session})
			req.AddCookie(&http.Cookie{Name: security.DefaultCSRFCookieName, Value: token})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			wantCalls := before
			if tt.want == http.StatusSeeOther {
				wantCalls++
			}
			if recorder.Code != tt.want || accounts.createCalls != wantCalls {
				t.Fatalf("status/calls = %d/%d, want %d/%d", recorder.Code, accounts.createCalls, tt.want, wantCalls)
			}
		})
	}
}

func TestClassTransferRoutesEnforceAuthenticationRoleAndCSRF(t *testing.T) {
	fake := &classTransferPageServiceFake{requestOut: service.RequestTransferOutput{TransferRequest: &domain.ClassTransferRequest{ID: uuid.New()}}}

	adminRouter, adminSession := newClassTransferRouterWithSession(t, domain.RoleAdmin, fake)
	unauthenticated := httptest.NewRecorder()
	adminRouter.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/class-transfer-requests/pending", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated pending list status = %d, want 401", unauthenticated.Code)
	}

	waliRouter, waliSession := newClassTransferRouterWithSession(t, domain.RoleWaliKelas, fake)
	waliReq := httptest.NewRequest(http.MethodGet, "/class-transfer-requests/pending", nil)
	waliReq.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: waliSession})
	waliRecorder := httptest.NewRecorder()
	waliRouter.ServeHTTP(waliRecorder, waliReq)
	if waliRecorder.Code != http.StatusForbidden || fake.listPendingCalls != 0 {
		t.Fatalf("Wali pending list status/calls = %d/%d, want 403/0", waliRecorder.Code, fake.listPendingCalls)
	}

	form := url.Values{
		"student_id": {uuid.NewString()}, "academic_year_id": {uuid.NewString()},
		"from_class_id": {uuid.NewString()}, "to_class_id": {uuid.NewString()},
	}
	missingCSRF := httptest.NewRequest(http.MethodPost, "/class-transfer-requests", strings.NewReader(form.Encode()))
	missingCSRF.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	missingCSRF.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: adminSession})
	missingCSRFRecorder := httptest.NewRecorder()
	adminRouter.ServeHTTP(missingCSRFRecorder, missingCSRF)
	if missingCSRFRecorder.Code != http.StatusForbidden || fake.requestCalls != 0 {
		t.Fatalf("POST without CSRF status/calls = %d/%d, want 403/0", missingCSRFRecorder.Code, fake.requestCalls)
	}

	waliCSRF, err := security.GenerateCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	form.Set("csrf_token", waliCSRF)
	validReq := httptest.NewRequest(http.MethodPost, "/class-transfer-requests", strings.NewReader(form.Encode()))
	validReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	validReq.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: waliSession})
	validReq.AddCookie(&http.Cookie{Name: security.DefaultCSRFCookieName, Value: waliCSRF})
	validRecorder := httptest.NewRecorder()
	waliRouter.ServeHTTP(validRecorder, validReq)
	if validRecorder.Code != http.StatusSeeOther || fake.requestCalls != 1 {
		t.Fatalf("POST with valid auth/CSRF status/calls = %d/%d, want 303/1: %s", validRecorder.Code, fake.requestCalls, validRecorder.Body.String())
	}
	if fake.requestInput.Actor.Role != domain.RoleWaliKelas || fake.requestInput.Actor.Username != "class-transfer-test-user" {
		t.Fatalf("request actor = %+v, want authenticated Wali Kelas", fake.requestInput.Actor)
	}
}
