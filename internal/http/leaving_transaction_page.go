package http

import (
	"errors"
	"html/template"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type lateRequestRow struct {
	ID, Year, Student, NIS, Type, Amount, Date, Reason, Evidence, Status, RequestedAt, Decision string
	CanDecide, CanExecute                                                                       bool
}
type lateRequestPageVM struct {
	Title, Heading, Description, Role, Username, ActiveNav, CSRFToken, Error, Notice, Status string
	Rows                                                                                     []lateRequestRow
}
type LeavingTransactionRequestPageHandler struct {
	service   service.LeavingTransactionRequestService
	templates *template.Template
}

func NewLeavingTransactionRequestPageHandler(s service.LeavingTransactionRequestService, t *template.Template) (*LeavingTransactionRequestPageHandler, error) {
	if s == nil || t == nil {
		return nil, service.ErrInvalidDependency
	}
	return &LeavingTransactionRequestPageHandler{service: s, templates: t}, nil
}

func (h *LeavingTransactionRequestPageHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	status := domain.LeavingTransactionRequestStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		status = domain.LeavingTransactionPending
	}
	out, err := h.service.List(r.Context(), service.ListLeavingTransactionRequestsInput{Actor: actor, Status: status})
	if err != nil {
		log.Printf("list leaving transaction requests: %v", err)
		httperror.Write(w, err)
		return
	}
	vm := lateRequestPageVM{Title: "Transaksi Terlewat", Heading: "Permintaan Transaksi Terlewat", Description: "Transaksi sebelum siswa pindah hanya dicatat setelah Admin menyetujui rincian dan bukti.", Role: string(actor.Role), Username: actor.Username, ActiveNav: "late-transactions", Status: string(status), Rows: []lateRequestRow{}}
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	}
	for _, x := range out.Rows {
		nis := ""
		if x.Student.NIS != nil {
			nis = *x.Student.NIS
		}
		row := lateRequestRow{ID: x.Request.ID.String(), Year: x.Year.Name, Student: x.Student.Name, NIS: nis, Type: transactionTypeLabel(x.Request.Type), Amount: formatRupiah(x.Request.Amount), Date: x.Request.TransactionDate.Format("2006-01-02"), Reason: x.Request.Reason, Evidence: x.Request.EvidenceReference, Status: interfaceStatusLabel(string(x.Request.Status)), RequestedAt: x.Request.RequestedAt.Format("2006-01-02 15:04"), Decision: x.Request.DecisionReason, CanDecide: actor.Role == domain.RoleAdmin && x.Request.Status == domain.LeavingTransactionPending, CanExecute: actor.Role == domain.RoleWaliKelas && x.Request.Status == domain.LeavingTransactionApproved}
		vm.Rows = append(vm.Rows, row)
	}
	if r.URL.Query().Get("notice") == "requested" {
		vm.Notice = "Permintaan tercatat dan menunggu pemeriksaan Admin."
	}
	if r.URL.Query().Get("notice") == "decided" {
		vm.Notice = "Keputusan Admin tersimpan."
	}
	if r.URL.Query().Get("notice") == "completed" {
		vm.Notice = "Transaksi yang disetujui berhasil dicatat ke buku tabungan."
	}
	if err := h.templates.ExecuteTemplate(w, "leaving-transaction-requests", vm); err != nil {
		log.Printf("render leaving transaction requests page: %v", err)
		http.Error(w, localizedStatusMessage(500), 500)
	}
}
func (h *LeavingTransactionRequestPageHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if err := parseLateRequestForm(w, r, "type", "amount", "transaction_date", "reason", "evidence_reference", "csrf_token"); err != nil {
		http.Error(w, "Form permintaan tidak valid", 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("account_id"))
	if err != nil {
		http.Error(w, "Rekening tidak valid.", 400)
		return
	}
	amount, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("amount")), 10, 64)
	if err != nil || amount <= 0 {
		http.Error(w, "Nominal tidak valid", 400)
		return
	}
	date, err := time.Parse("2006-01-02", r.FormValue("transaction_date"))
	if err != nil {
		http.Error(w, "Tanggal tidak valid", 400)
		return
	}
	typ := domain.TransactionType(strings.TrimSpace(r.FormValue("type")))
	if typ != domain.TransactionDeposit && typ != domain.TransactionWithdrawal {
		http.Error(w, "Jenis transaksi tidak valid", 400)
		return
	}
	_, err = h.service.Create(r.Context(), service.CreateLeavingTransactionRequestInput{Actor: actor, SavingsAccountID: id, Type: typ, Amount: domain.Money(amount), TransactionDate: date, Reason: r.FormValue("reason"), EvidenceReference: r.FormValue("evidence_reference")})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/leaving-transaction-requests?notice=requested", http.StatusSeeOther)
}
func (h *LeavingTransactionRequestPageHandler) Decide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if err := parseLateRequestForm(w, r, "decision", "reason", "csrf_token"); err != nil {
		http.Error(w, "Form keputusan tidak valid", 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("request_id"))
	if err != nil {
		http.Error(w, "Permintaan tidak valid.", 400)
		return
	}
	decision := r.FormValue("decision")
	if decision != "approve" && decision != "reject" {
		http.Error(w, "Keputusan tidak valid", 400)
		return
	}
	err = h.service.Decide(r.Context(), service.DecideLeavingTransactionRequestInput{Actor: actor, RequestID: id, Approve: decision == "approve", Reason: r.FormValue("reason")})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/leaving-transaction-requests?notice=decided", http.StatusSeeOther)
}

func parseLateRequestForm(w http.ResponseWriter, r *http.Request, allowed ...string) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		return errors.New("invalid content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		return err
	}
	valid := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		valid[key] = true
	}
	for key, values := range r.PostForm {
		if !valid[key] || len(values) != 1 {
			return errors.New("invalid form field")
		}
	}
	return nil
}
func (h *LeavingTransactionRequestPageHandler) Execute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	id, err := uuid.Parse(r.PathValue("request_id"))
	if err != nil {
		http.Error(w, "Permintaan tidak valid.", 400)
		return
	}
	if _, err = h.service.Execute(r.Context(), service.ExecuteLeavingTransactionRequestInput{Actor: actor, RequestID: id}); err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/leaving-transaction-requests?notice=completed", http.StatusSeeOther)
}
