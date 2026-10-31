package http

import (
	"bytes"
	"errors"
	"html/template"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

const maxSettlementFormBytes = 16 << 10

type SettlementReadinessRowVM struct{ StudentName, NIS, Class, Wali, Status, Balance, AccountURL, SettlementURL, PreviewURL string }
type SettlementYearOptionVM struct{ ID, Name string }
type SettlementGroupVM struct {
	Wali, Class                              string
	Status                                   string
	Outstanding, AwaitingHandover, Completed int
}
type SettlementPageVM struct {
	Title, Heading, Description, Role, Username, ActiveNav, CSRFToken                                                                               string
	AcademicYearID, YearName, YearStatus, Error, Notice                                                                                             string
	Total, Outstanding, Settled, AwaitingHandover, Completed, Unattributed, PendingTransfers                                                        int
	Ready                                                                                                                                           bool
	PendingLateTransactions                                                                                                                         bool
	Rows                                                                                                                                            []SettlementReadinessRowVM
	Years                                                                                                                                           []SettlementYearOptionVM
	Groups                                                                                                                                          []SettlementGroupVM
	IsDetail                                                                                                                                        bool
	CanSettle                                                                                                                                       bool
	IsPreview                                                                                                                                       bool
	SettlementID, AccountID, StudentName, SettlementType, Amount, ExecutedBy, ExecutedAt, Delivered, CashStatus, ReconciliationDirection            string
	ExpectedBalance                                                                                                                                 string
	PreviewedAt                                                                                                                                     string
	ReadinessLabel, TransferURL                                                                                                                     string
	CanHandover                                                                                                                                     bool
	CanRequestReopen, CanExecuteReopen, CanDecideReopen                                                                                             bool
	ReopenRequestID, ReopenRequestStatus, ReopenRequestReason, ReopenRequestedAt, ReopenDecisionReason, ReopenDecidedAt, ReopenDecidedBy, YearState string
	Movements                                                                                                                                       []SettlementMovementVM
	History                                                                                                                                         []SettlementHistoryVM
}
type SettlementMovementVM struct{ Kind, Amount, Actor, At, Reference string }
type SettlementHistoryVM struct{ Type, Amount, Status, ExecutedBy, ExecutedAt string }
type SettlementPageHandler struct {
	service   service.SettlementService
	years     service.AcademicYearService
	templates *template.Template
}

func NewSettlementPageHandler(s service.SettlementService, y service.AcademicYearService, t *template.Template) (*SettlementPageHandler, error) {
	if s == nil || y == nil || t == nil {
		return nil, service.ErrInvalidDependency
	}
	return &SettlementPageHandler{service: s, years: y, templates: t}, nil
}
func (h *SettlementPageHandler) Overview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	out, err := h.years.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: 250}})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("academic_year_id"))
	var yearID uuid.UUID
	if id != "" {
		yearID, err = uuid.Parse(id)
		if err != nil || yearID == uuid.Nil {
			http.Error(w, localizedStatusMessage(400), 400)
			return
		}
	} else {
		for _, y := range out.AcademicYears {
			if y != nil && y.IsClosing() {
				yearID = y.ID
				break
			}
		}
	}
	heading, description, activeNav := "Proses Tutup Buku Tabungan", "Periksa rekening dan penyerahan uang dalam cakupan kelas Anda.", "savings"
	if actor.Role == domain.RoleAdmin {
		heading, description, activeNav = "Penutupan Tahun", "Lihat wali kelas yang masih memiliki rekening atau penyerahan uang yang menghambat penutupan tahun ajaran.", "year-end"
	}
	vm := SettlementPageVM{Title: heading, Heading: heading, Description: description, Role: string(actor.Role), Username: actor.Username, ActiveNav: activeNav, AcademicYearID: id}
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	}
	for _, y := range out.AcademicYears {
		if y != nil {
			vm.Years = append(vm.Years, SettlementYearOptionVM{ID: y.ID.String(), Name: y.Name})
			if y.ID == yearID {
				vm.YearName = y.Name
				vm.YearStatus = interfaceStatusLabel(string(y.Status))
			}
		}
	}
	if yearID != uuid.Nil {
		vm.AcademicYearID = yearID.String()
		ready, e := h.service.YearReadiness(r.Context(), service.YearSettlementReadinessInput{Actor: actor, AcademicYearID: yearID})
		if e != nil {
			httperror.Write(w, e)
			return
		}
		vm.Total, vm.Outstanding, vm.Settled, vm.AwaitingHandover, vm.Completed, vm.Unattributed, vm.PendingTransfers, vm.Ready = ready.Total, ready.Outstanding, ready.Settled, ready.AwaitingHandover, ready.Completed, ready.Unattributed, ready.PendingTransfers, ready.Ready
		vm.PendingLateTransactions = ready.PendingLateTransactions
		if ready.PendingLateTransactions {
			vm.Notice = "Masih ada transaksi terlewat yang menunggu keputusan Admin atau pencatatan Wali. Selesaikan melalui menu Transaksi Terlewat sebelum penyelesaian rekening atau penutupan tahun."
		}
		vm.ReadinessLabel = "Kesiapan tahun ajaran"
		if actor.Role == domain.RoleWaliKelas {
			vm.ReadinessLabel = "Kesiapan cakupan Anda"
		}
		vm.TransferURL = "/class-transfer-requests/pending?academic_year_id=" + yearID.String()
		for _, x := range ready.Rows {
			row := SettlementReadinessRowVM{StudentName: x.Student.Name, NIS: stringValue(x.Student.NIS), Class: x.ClassName, Wali: x.WaliName, Status: interfaceStatusLabel(string(x.Account.Status)), Balance: formatRupiah(x.CurrentBalance), AccountURL: "/savings/" + x.Account.ID.String()}
			if x.AttributionIssue {
				row.Class = "Perlu pemeriksaan data"
			}
			if x.Account.IsOpen() {
				row.Balance = formatRupiah(x.CurrentBalance)
				if vm.YearStatus == interfaceStatusLabel(string(domain.AcademicYearClosing)) {
					row.PreviewURL = "/savings/" + x.Account.ID.String() + "/settlement-preview"
				}
			} else if x.Settlement != nil {
				row.Balance = formatRupiah(x.CurrentBalance)
				row.SettlementURL = "/settlements/" + x.Settlement.ID.String()
				if x.Settlement.Amount != x.CashDelivered {
					row.Status = "Menunggu penyerahan"
				} else {
					row.Status = "Selesai"
				}
			}
			if x.AttributionIssue {
				row.Wali = "Data penanggung jawab ambigu"
			} else if x.WaliID == nil || x.ClassID == nil {
				row.Wali = "Belum ada penanggung jawab"
			}
			vm.Rows = append(vm.Rows, row)
		}
		groups := map[string]*SettlementGroupVM{}
		for _, x := range ready.Rows {
			wali, class := x.WaliName, x.ClassName
			if x.AttributionIssue {
				wali, class = "Data penanggung jawab ambigu", "Perlu pemeriksaan data"
			} else if x.WaliID == nil || x.ClassID == nil {
				wali, class = "Belum ada penanggung jawab", "Belum terpetakan"
			}
			key := wali + "\x00" + class
			g := groups[key]
			if g == nil {
				g = &SettlementGroupVM{Wali: wali, Class: class}
				groups[key] = g
			}
			if x.Account.IsOpen() {
				g.Outstanding++
			} else if x.Settlement != nil && x.Settlement.Amount != x.CashDelivered {
				g.AwaitingHandover++
			} else {
				g.Completed++
			}
		}
		for _, g := range groups {
			g.Status = "Siap"
			if g.Outstanding > 0 || g.AwaitingHandover > 0 {
				g.Status = "Menghambat penutupan"
			}
			vm.Groups = append(vm.Groups, *g)
		}
		sort.Slice(vm.Groups, func(i, j int) bool {
			if vm.Groups[i].Wali == vm.Groups[j].Wali {
				return vm.Groups[i].Class < vm.Groups[j].Class
			}
			return vm.Groups[i].Wali < vm.Groups[j].Wali
		})
	}
	h.render(w, vm)
}
func (h *SettlementPageHandler) Preview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	id, err := uuid.Parse(r.PathValue("account_id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	p, err := h.service.PreviewYearEnd(r.Context(), service.PreviewYearEndInput{Actor: actor, SavingsAccountID: id})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	h.renderPreview(w, r, actor, id, p, "", http.StatusOK)
}

func (h *SettlementPageHandler) PreviewStudentLeaving(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(r.PathValue("account_id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	preview, err := h.service.PreviewStudentLeaving(r.Context(), service.PreviewStudentLeavingInput{Actor: actor, SavingsAccountID: id})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	h.renderStudentLeavingPreview(w, r, actor, id, preview, "", http.StatusOK)
}

func (h *SettlementPageHandler) SettleStudentLeaving(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err := parseSettlementForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(r.PathValue("account_id"))
	amount, amountErr := strconv.ParseInt(r.PostForm.Get("expected_balance"), 10, 64)
	if err != nil || id == uuid.Nil || amountErr != nil || amount < 0 {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	out, err := h.service.SettleStudentLeaving(r.Context(), service.SettleStudentLeavingInput{Actor: actor, SavingsAccountID: id, ExpectedBalance: domain.Money(amount)})
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			preview, previewErr := h.service.PreviewStudentLeaving(r.Context(), service.PreviewStudentLeavingInput{Actor: actor, SavingsAccountID: id})
			if previewErr == nil {
				h.renderStudentLeavingPreview(w, r, actor, id, preview, "Saldo berubah sejak pratinjau. Periksa saldo terbaru dan konfirmasikan kembali.", http.StatusConflict)
				return
			}
		}
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/settlements/"+out.Settlement.ID.String()+"?notice=settled", http.StatusSeeOther)
}
func (h *SettlementPageHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if err := parseSettlementForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("account_id"))
	amount, e := strconv.ParseInt(r.PostForm.Get("expected_balance"), 10, 64)
	if err != nil || id == uuid.Nil || e != nil || amount < 0 {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	out, err := h.service.SettleYearEnd(r.Context(), service.SettleYearEndInput{Actor: actor, SavingsAccountID: id, ExpectedBalance: domain.Money(amount)})
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			preview, previewErr := h.service.PreviewYearEnd(r.Context(), service.PreviewYearEndInput{Actor: actor, SavingsAccountID: id})
			if previewErr == nil {
				h.renderPreview(w, r, actor, id, preview, "Saldo berubah sejak pratinjau. Periksa saldo terbaru dan konfirmasikan kembali.", http.StatusConflict)
				return
			}
		}
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/settlements/"+out.Settlement.ID.String()+"?notice=settled", http.StatusSeeOther)
}
func (h *SettlementPageHandler) Handover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if err := parseSettlementForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("settlement_id"))
	if err != nil || id == uuid.Nil || r.PostForm.Get("confirm_handover") != "1" {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	ref := strings.TrimSpace(r.PostForm.Get("receipt_reference"))
	var refPtr *string
	if ref != "" {
		refPtr = &ref
	}
	if _, err := h.service.ConfirmHandover(r.Context(), service.ConfirmSettlementHandoverInput{Actor: actor, SettlementID: id, Confirm: true, ReceiptReference: refPtr}); err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/settlements/"+id.String()+"?notice=handover_confirmed", http.StatusSeeOther)
}
func (h *SettlementPageHandler) Detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	id, err := uuid.Parse(r.PathValue("settlement_id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	out, err := h.service.Get(r.Context(), service.GetSettlementInput{Actor: actor, SettlementID: id})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	s := out.Settlement
	executedBy := out.ActorNames[s.ExecutedBy]
	if executedBy == "" {
		executedBy = s.ExecutedBy.String()
	}
	vm := SettlementPageVM{Title: "Detail Penyelesaian Tabungan", Heading: "Detail Penyelesaian Tabungan", Description: "Bukti penyelesaian tabungan dan penyerahan uang.", Role: string(actor.Role), Username: actor.Username, ActiveNav: "savings", IsDetail: true, SettlementID: s.ID.String(), SettlementType: string(s.Type), Amount: formatRupiah(s.Amount), ExecutedBy: executedBy, ExecutedAt: s.ExecutedAt.Format(time.RFC3339)}
	if out.Account != nil {
		year, e := h.years.Get(r.Context(), service.GetAcademicYearInput{Actor: actor, AcademicYearID: out.Account.AcademicYearID})
		if e == nil && year.AcademicYear != nil {
			vm.YearState = interfaceStatusLabel(string(year.AcademicYear.Status))
			vm.CanRequestReopen = actor.Role == domain.RoleWaliKelas && year.AcademicYear.IsClosed() && out.Account.IsSettled() && s.Type == domain.SettlementRegularYearEnd
			vm.CanExecuteReopen = actor.Role == domain.RoleWaliKelas && year.AcademicYear.IsClosed() && out.Account.IsSettled()
		}
	}
	if out.ReopenRequest != nil {
		vm.ReopenRequestID = out.ReopenRequest.ID.String()
		vm.ReopenRequestStatus = interfaceStatusLabel(string(out.ReopenRequest.Status))
		vm.ReopenRequestReason = out.ReopenRequest.Reason
		if name := out.ActorNames[out.ReopenRequest.RequestedBy]; name != "" {
			vm.ReopenRequestReason = name + ": " + vm.ReopenRequestReason
		}
		vm.ReopenRequestedAt = out.ReopenRequest.RequestedAt.Format(time.RFC3339)
		if out.ReopenRequest.DecisionReason != nil {
			vm.ReopenDecisionReason = *out.ReopenRequest.DecisionReason
		}
		if out.ReopenRequest.DecidedAt != nil {
			vm.ReopenDecidedAt = out.ReopenRequest.DecidedAt.Format(time.RFC3339)
		}
		if out.ReopenRequest.DecidedBy != nil {
			vm.ReopenDecidedBy = out.ActorNames[*out.ReopenRequest.DecidedBy]
		}
		vm.CanRequestReopen = false
		vm.CanExecuteReopen = vm.CanExecuteReopen && out.ReopenRequest.Status == domain.SettlementReopenApproved
		vm.CanDecideReopen = actor.Role == domain.RoleAdmin && out.ReopenRequest.Status == domain.SettlementReopenPending
	}
	if out.Student != nil {
		vm.StudentName = out.Student.Name
	}
	var delivered domain.Money
	for _, m := range out.Movements {
		if m.Kind == domain.SettlementCashHandover {
			delivered += m.Amount
		} else {
			delivered -= m.Amount
		}
		ref := ""
		if m.ReceiptReference != nil {
			ref = *m.ReceiptReference
		}
		movementActor := out.ActorNames[m.ConfirmedBy]
		if movementActor == "" {
			movementActor = m.ConfirmedBy.String()
		}
		vm.Movements = append(vm.Movements, SettlementMovementVM{Kind: interfaceSettlementMovementLabel(string(m.Kind)), Amount: formatRupiah(m.Amount), Actor: movementActor, At: m.ConfirmedAt.Format(time.RFC3339), Reference: ref})
	}
	for _, item := range out.History {
		name := out.ActorNames[item.ExecutedBy]
		if name == "" {
			name = item.ExecutedBy.String()
		}
		vm.History = append(vm.History, SettlementHistoryVM{Type: interfaceSettlementTypeLabel(string(item.Type)), Amount: formatRupiah(item.Amount), Status: interfaceStatusLabel(string(item.Status)), ExecutedBy: name, ExecutedAt: item.ExecutedAt.Format(time.RFC3339)})
	}
	vm.Delivered = formatRupiah(delivered)
	vm.CashStatus = "Terkonfirmasi"
	if s.Amount == 0 {
		vm.CashStatus = "Tidak diperlukan karena saldo nol"
	} else if delivered != s.Amount {
		vm.CashStatus = "Rekonsiliasi penyerahan diperlukan"
		vm.CanHandover = actor.Role == domain.RoleWaliKelas
		if delivered < s.Amount {
			vm.ReconciliationDirection = "penyerahan tambahan"
		} else {
			vm.ReconciliationDirection = "pengembalian selisih"
		}
	}
	if r.URL.Query().Get("notice") == "settled" {
		vm.Notice = "Penyelesaian tabungan berhasil dicatat."
	}
	if r.URL.Query().Get("notice") == "handover_confirmed" {
		vm.Notice = "Penyerahan uang berhasil dikonfirmasi."
	}
	if r.URL.Query().Get("notice") == "reopen_requested" {
		vm.Notice = "Permintaan pembukaan kembali dikirim untuk persetujuan Admin."
	}
	if r.URL.Query().Get("notice") == "reopen_decided" {
		vm.Notice = "Keputusan Admin berhasil dicatat."
	}
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	}
	h.render(w, vm)
}

func (h *SettlementPageHandler) RequestReopen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if err := parseSettlementForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("settlement_id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	_, err = h.service.RequestReopen(r.Context(), service.RequestSettlementReopenInput{Actor: actor, SettlementID: id, Reason: r.PostForm.Get("reason")})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/settlements/"+id.String()+"?notice=reopen_requested", http.StatusSeeOther)
}
func (h *SettlementPageHandler) DecideReopen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if err := parseSettlementForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("request_id"))
	decision := r.PostForm.Get("decision")
	if err != nil || id == uuid.Nil || (decision != "approve" && decision != "reject") {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	out, err := h.service.DecideReopen(r.Context(), service.DecideSettlementReopenInput{Actor: actor, RequestID: id, Approve: decision == "approve", Reason: r.PostForm.Get("reason")})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/settlements/"+out.Request.SavingsSettlementID.String()+"?notice=reopen_decided", http.StatusSeeOther)
}
func (h *SettlementPageHandler) Reopen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if err := parseSettlementForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("settlement_id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	out, err := h.service.Reopen(r.Context(), service.ReopenSettlementInput{Actor: actor, SettlementID: id})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/savings/"+out.Account.ID.String(), http.StatusSeeOther)
}
func parseSettlementForm(w http.ResponseWriter, r *http.Request) error {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/x-www-form-urlencoded" {
		return errors.New("invalid content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSettlementFormBytes)
	return r.ParseForm()
}
func (h *SettlementPageHandler) render(w http.ResponseWriter, vm SettlementPageVM) {
	if err := h.templates.ExecuteTemplate(w, "settlement-page", vm); err != nil {
		http.Error(w, localizedStatusMessage(500), 500)
	}
}

func (h *SettlementPageHandler) renderPreview(w http.ResponseWriter, r *http.Request, actor service.Actor, id uuid.UUID, preview service.PreviewYearEndOutput, message string, status int) {
	token, ok := middleware.CSRFToken(r)
	if !ok && r.Method == http.MethodPost {
		token = r.PostForm.Get("csrf_token")
		ok = token != ""
	}
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm := SettlementPageVM{
		Title: "Pratinjau Penyelesaian Tabungan", Heading: "Konfirmasi Penyelesaian Tabungan",
		Description: "Tinjau saldo buku penuh sebelum menyelesaikan rekening.",
		Role:        string(actor.Role), Username: actor.Username, ActiveNav: "savings",
		IsPreview: true, CanSettle: actor.Role == domain.RoleWaliKelas,
		AccountID: id.String(), ExpectedBalance: strconv.FormatInt(int64(preview.Balance), 10),
		Amount: formatRupiah(preview.Balance), StudentName: preview.Student.Name,
		CSRFToken: token, PreviewedAt: preview.PreviewedAt.Format(time.RFC3339), Error: message,
	}
	var body bytes.Buffer
	if err := h.templates.ExecuteTemplate(&body, "settlement-page", vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}

func (h *SettlementPageHandler) renderStudentLeavingPreview(w http.ResponseWriter, r *http.Request, actor service.Actor, id uuid.UUID, preview service.PreviewYearEndOutput, message string, status int) {
	token, ok := middleware.CSRFToken(r)
	if !ok && r.Method == http.MethodPost {
		token = r.PostForm.Get("csrf_token")
		ok = token != ""
	}
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm := SettlementPageVM{
		Title: "Penyelesaian Siswa Pindah Sekolah", Heading: "Penyelesaian Tabungan Pindah Sekolah",
		Description: "Tinjau saldo buku tabungan siswa yang pindah sebelum menyelesaikannya.",
		Role:        string(actor.Role), Username: actor.Username, ActiveNav: "savings", IsPreview: true,
		CanSettle: actor.Role == domain.RoleWaliKelas, SettlementType: string(domain.SettlementStudentLeaving),
		AccountID: id.String(), ExpectedBalance: strconv.FormatInt(int64(preview.Balance), 10),
		Amount: formatRupiah(preview.Balance), StudentName: preview.Student.Name,
		CSRFToken: token, PreviewedAt: preview.PreviewedAt.Format(time.RFC3339), Error: message,
	}
	var body bytes.Buffer
	if err := h.templates.ExecuteTemplate(&body, "settlement-page", vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}

func validateSettlementRequestForm(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := parseSettlementForm(w, r); err != nil {
			http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
