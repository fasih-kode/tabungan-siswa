package http

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

const maxClassTransferFormBytes = 16 << 10

type ClassTransferRow struct {
	ID, StudentID, FromClassID, ToClassID string
	Status, RequestedAt, RejectionReason  string
	CanReview                             bool
}

type ClassTransferPageViewModel struct {
	Pagination                                                        PaginationViewModel
	Title, Heading, Description, CSRFToken, Role, Username, ActiveNav string
	StudentID, AcademicYearID, FromClassID, FromClassName, ToClassID  string
	TransferRequestID, EffectiveDate, RejectionReason                 string
	ListURL                                                           string
	Error, StatusMessage, RequestUnavailableReason                    string
	Rows                                                              []ClassTransferRow
	TargetClasses                                                     []TransferClassOption
	Page, Limit, PreviousPage, NextPage                               int
	HasPrevious, HasNext, IsDetail, IsStudentList, CanRequest         bool
}

type TransferClassOption struct{ ID, Name string }

type ClassTransferPageHandler struct {
	transfers service.ClassTransferService
	templates *template.Template
}

func NewClassTransferPageHandler(transfers service.ClassTransferService, templates *template.Template) (*ClassTransferPageHandler, error) {
	if transfers == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &ClassTransferPageHandler{transfers: transfers, templates: templates}, nil
}

func (h *ClassTransferPageHandler) ListStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	studentID, err := parsePathUUID(r.PathValue("student_id"))
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	yearID, err := parseQueryUUID(r, "academic_year_id")
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	page, limit, err := parsePageOptions(r)
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	output, err := h.transfers.ListByStudent(r.Context(), service.ListStudentTransfersInput{
		Actor: actor, StudentID: studentID, AcademicYearID: yearID,
		Options: service.ListOptions{Limit: limit + 1, Offset: (page - 1) * limit, IncludeTotal: true},
	})
	if err != nil {
		httpError(w, err)
		return
	}
	vm, ok := h.baseViewModel(r, actor)
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm.Heading, vm.Description, vm.IsStudentList = "Perpindahan Kelas Siswa", "Riwayat dan pengajuan perpindahan kelas.", true
	vm.StudentID, vm.AcademicYearID = studentID.String(), yearID.String()
	if actor.Role == domain.RoleWaliKelas {
		options, optionsErr := h.transfers.RequestOptions(r.Context(), service.RequestTransferOptionsInput{Actor: actor, StudentID: studentID, AcademicYearID: yearID})
		if optionsErr != nil {
			httpError(w, optionsErr)
			return
		}
		vm.CanRequest = options.CanRequest
		vm.RequestUnavailableReason = options.UnavailableReason
		vm.FromClassID = options.FromClassID.String()
		vm.FromClassName = options.FromClassName
		for _, class := range options.TargetClasses {
			if class != nil {
				vm.TargetClasses = append(vm.TargetClasses, TransferClassOption{ID: class.ID.String(), Name: class.Name})
			}
		}
	}
	vm.ListURL = "/students/" + studentID.String() + "/class-transfer-requests?academic_year_id=" + url.QueryEscape(yearID.String())
	vm.Page, vm.Limit = page, limit
	vm.Rows, vm.HasNext = transferRows(output.TransferRequests, limit)
	vm.HasPrevious, vm.PreviousPage, vm.NextPage = page > 1, page-1, page+1
	vm.Pagination = newPagination(r, page, output.Total, len(vm.Rows), vm.HasNext)
	if redirectMissingPage(w, r, vm.Pagination) {
		return
	}
	if err := h.templates.ExecuteTemplate(w, "class-transfers", vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *ClassTransferPageHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if actor.Role != domain.RoleAdmin {
		http.Error(w, localizedStatusMessage(http.StatusForbidden), http.StatusForbidden)
		return
	}
	yearValue := strings.TrimSpace(r.URL.Query().Get("academic_year_id"))
	if yearValue == "" {
		vm, ok := h.baseViewModel(r, actor)
		if !ok {
			http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		vm.Heading, vm.Description = "Permintaan Pindah Kelas", "Pilih tahun ajaran untuk menampilkan permintaan yang menunggu keputusan."
		vm.Page, vm.Limit = 1, defaultAcademicYearPageSize
		if redirectMissingPage(w, r, vm.Pagination) {
			return
		}
		if err := h.templates.ExecuteTemplate(w, "class-transfers", vm); err != nil {
			http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	yearID, err := parseUUID(yearValue)
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	page, limit, err := parsePageOptions(r)
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	output, err := h.transfers.ListPending(r.Context(), service.ListPendingTransfersInput{
		Actor: actor, AcademicYearID: yearID,
		Options: service.ListOptions{Limit: limit + 1, Offset: (page - 1) * limit, IncludeTotal: true},
	})
	if err != nil {
		httpError(w, err)
		return
	}
	vm, ok := h.baseViewModel(r, actor)
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm.Heading, vm.Description = "Permintaan Pindah Kelas", "Tinjau permintaan pindah kelas yang masih menunggu keputusan."
	vm.AcademicYearID, vm.Page, vm.Limit = yearID.String(), page, limit
	vm.ListURL = "/class-transfer-requests/pending?academic_year_id=" + url.QueryEscape(yearID.String())
	vm.Rows, vm.HasNext = transferRows(output.TransferRequests, limit)
	vm.HasPrevious, vm.PreviousPage, vm.NextPage = page > 1, page-1, page+1
	vm.Pagination = newPagination(r, page, output.Total, len(vm.Rows), vm.HasNext)
	if redirectMissingPage(w, r, vm.Pagination) {
		return
	}
	if err := h.templates.ExecuteTemplate(w, "class-transfers", vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *ClassTransferPageHandler) Detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	requestID, err := parsePathUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	output, err := h.transfers.Get(r.Context(), service.GetTransferInput{Actor: actor, TransferRequestID: requestID})
	if err != nil {
		httpError(w, err)
		return
	}
	vm, ok := h.baseViewModel(r, actor)
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm.Heading, vm.Description, vm.IsDetail = "Detail Pindah Kelas", "Detail dan status permintaan pindah kelas.", true
	vm.AcademicYearID = output.TransferRequest.AcademicYearID.String()
	if actor.Role == domain.RoleAdmin {
		vm.ListURL = "/class-transfer-requests/pending?academic_year_id=" + url.QueryEscape(vm.AcademicYearID)
	} else {
		vm.ListURL = "/students/" + output.TransferRequest.StudentID.String() + "/class-transfer-requests?academic_year_id=" + url.QueryEscape(vm.AcademicYearID)
	}
	vm.Rows, _ = transferRows([]*domain.ClassTransferRequest{output.TransferRequest}, 1)
	if len(vm.Rows) > 0 {
		vm.Rows[0].CanReview = actor.Role == domain.RoleAdmin && output.TransferRequest.IsPending()
	}
	vm.TransferRequestID = requestID.String()
	if err := h.templates.ExecuteTemplate(w, "class-transfers", vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *ClassTransferPageHandler) Request(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if actor.Role != domain.RoleWaliKelas {
		httpError(w, service.ErrForbidden)
		return
	}
	if err := parseClassTransferForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	studentID, err1 := parseFormUUID(r, "student_id")
	yearID, err2 := parseFormUUID(r, "academic_year_id")
	fromID, err3 := parseFormUUID(r, "from_class_id")
	toID, err4 := parseFormUUID(r, "to_class_id")
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	output, err := h.transfers.Request(r.Context(), service.RequestTransferInput{
		Actor: actor, StudentID: studentID, AcademicYearID: yearID,
		FromClassID: fromID, ToClassID: toID,
	})
	if err != nil {
		httpError(w, err)
		return
	}
	http.Redirect(w, r, "/class-transfer-requests/"+output.TransferRequest.ID.String(), http.StatusSeeOther)
}

func (h *ClassTransferPageHandler) Approve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	requestID, err := parsePathUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err := parseClassTransferForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	effectiveDate, err := time.Parse("2006-01-02", strings.TrimSpace(r.PostForm.Get("effective_date")))
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	_, err = h.transfers.Approve(r.Context(), service.ApproveTransferInput{Actor: actor, TransferRequestID: requestID, EffectiveDate: effectiveDate})
	if err != nil {
		httpError(w, err)
		return
	}
	http.Redirect(w, r, "/class-transfer-requests/"+requestID.String(), http.StatusSeeOther)
}

func (h *ClassTransferPageHandler) Reject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	requestID, err := parsePathUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err := parseClassTransferForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	_, err = h.transfers.Reject(r.Context(), service.RejectTransferInput{
		Actor: actor, TransferRequestID: requestID, RejectionReason: r.PostForm.Get("rejection_reason"),
	})
	if err != nil {
		httpError(w, err)
		return
	}
	http.Redirect(w, r, "/class-transfer-requests/"+requestID.String(), http.StatusSeeOther)
}

func (h *ClassTransferPageHandler) baseViewModel(r *http.Request, actor service.Actor) (ClassTransferPageViewModel, bool) {
	token, ok := middleware.CSRFToken(r)
	if !ok {
		return ClassTransferPageViewModel{}, false
	}
	return ClassTransferPageViewModel{Title: "Pindah Kelas", Role: string(actor.Role), Username: actor.Username, CSRFToken: token, ActiveNav: "class-transfers"}, true
}

func transferRows(requests []*domain.ClassTransferRequest, limit int) ([]ClassTransferRow, bool) {
	rows := make([]ClassTransferRow, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		row := ClassTransferRow{
			ID: request.ID.String(), StudentID: request.StudentID.String(),
			FromClassID: request.FromClassID.String(), ToClassID: request.ToClassID.String(),
			Status: interfaceStatusLabel(string(request.Status)), RequestedAt: request.CreatedAt.Format("2006-01-02 15:04"),
		}
		if request.RejectionReason != nil {
			row.RejectionReason = *request.RejectionReason
		}
		rows = append(rows, row)
	}
	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}
	return rows, hasNext
}

func parseClassTransferForm(w http.ResponseWriter, r *http.Request) error {
	contentType := strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0])
	if contentType != "application/x-www-form-urlencoded" {
		return errors.New("invalid content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxClassTransferFormBytes)
	return r.ParseForm()
}

func parsePathUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidID
	}
	return id, nil
}

func parseQueryUUID(r *http.Request, key string) (uuid.UUID, error) {
	return parseUUID(strings.TrimSpace(r.URL.Query().Get(key)))
}

func parseFormUUID(r *http.Request, key string) (uuid.UUID, error) {
	return parseUUID(strings.TrimSpace(r.PostForm.Get(key)))
}

func parseUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidID
	}
	return id, nil
}

func httpError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrEmptyRejectionReason) || errors.Is(err, domain.ErrInvalidDateRange) || errors.Is(err, domain.ErrInvalidID) {
		http.Error(w, "Permintaan tidak dapat diproses. Periksa kembali data yang dikirim.", httperror.Status(err))
		return
	}
	httperror.Write(w, err)
}
