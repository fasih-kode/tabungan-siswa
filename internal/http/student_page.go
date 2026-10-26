package http

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

const (
	maxStudentFormBytes   = 16 << 10
	maxStudentImportBytes = 5 << 20
	maxStudentImportRows  = 5000
)

type StudentOption struct{ ID, Name string }
type StudentHistoryRow struct{ AcademicYear, Class, From, To string }
type StudentRow struct{ ID, Name, NIS, NISN, Status, AcademicYear, Class, AcademicYearID, TransferURL string }
type StudentPageViewModel struct {
	Pagination                                                                           PaginationViewModel
	Title, Heading, Description, CSRFToken, Role, Username, Error, Notice, ActiveNav     string
	AcademicYearID, ClassID, StatusFilter, Search, ViewFilter                            string
	StudentID, Name, NIS, NISN, UserID, StudentStatus, PlacementYearID, PlacementClassID string
	TransferURL                                                                          string
	Page, Limit, PreviousPage, NextPage                                                  int
	HasPrevious, HasNext, IsDetail, IsEdit, IsAdmin, IsImport                            bool
	Rows                                                                                 []StudentRow
	PlacementClasses                                                                     []StudentClassOption
	Years, Classes, Users                                                                []StudentOption
	History                                                                              []StudentHistoryRow
	ImportErrors                                                                         []service.StudentImportIssue
}

type StudentPageHandler struct {
	students  service.StudentService
	years     service.AcademicYearService
	classes   service.ClassService
	users     service.UserService
	templates *template.Template
}

func NewStudentPageHandler(students service.StudentService, years service.AcademicYearService, classes service.ClassService, users service.UserService, templates *template.Template) (*StudentPageHandler, error) {
	if students == nil || years == nil || classes == nil || users == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &StudentPageHandler{students: students, years: years, classes: classes, users: users, templates: templates}, nil
}

func (h *StudentPageHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if actor.Role == domain.RoleSiswa {
		out, err := h.students.GetByUserID(r.Context(), service.GetStudentByUserIDInput{Actor: actor, UserID: actor.UserID})
		if err != nil {
			httperror.Write(w, err)
			return
		}
		http.Redirect(w, r, "/students/"+out.Student.ID.String(), http.StatusSeeOther)
		return
	}
	page, limit, err := parsePageOptions(r)
	if err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	yearID, classID := strings.TrimSpace(r.URL.Query().Get("academic_year_id")), strings.TrimSpace(r.URL.Query().Get("class_id"))
	search, status := strings.TrimSpace(r.URL.Query().Get("q")), strings.TrimSpace(r.URL.Query().Get("status"))
	view := strings.TrimSpace(r.URL.Query().Get("view"))
	if actor.Role == domain.RoleAdmin && view == "" {
		view = "all"
	}
	if actor.Role == domain.RoleWaliKelas {
		view = "class"
	}
	vm := h.baseVM(actor, "Daftar Siswa", "Kelola siswa dan penempatan kelas.")
	vm.AcademicYearID, vm.ClassID, vm.Search, vm.StatusFilter, vm.ViewFilter = yearID, classID, search, status, view
	if actor.Role == domain.RoleWaliKelas && r.URL.Query().Get("transfer") == "1" {
		vm.ActiveNav = "class-transfers"
		vm.Notice = "Pilih siswa, lalu pilih Ajukan pindah kelas untuk mengajukan perpindahan kelas."
	}
	if actor.Role == domain.RoleAdmin && r.URL.Query().Get("notice") == "imported" {
		count, parseErr := strconv.Atoi(r.URL.Query().Get("count"))
		if parseErr == nil && count > 0 {
			vm.Notice = strconv.Itoa(count) + " siswa berhasil diimpor."
		}
	} else if actor.Role == domain.RoleAdmin && r.URL.Query().Get("notice") == "created" {
		vm.Notice = "Siswa berhasil ditambahkan beserta penempatan kelasnya."
	}
	vm.Page, vm.Limit = page, limit
	if err := h.loadOptions(r, actor, &vm, yearID, false); err != nil {
		httperror.Write(w, err)
		return
	}
	if actor.Role == domain.RoleWaliKelas && yearID == "" {
		yearID = vm.AcademicYearID
	}
	if view != "all" && view != "unplaced" && view != "class" {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	if (yearID != "" && parseOptionalUUID(yearID) == uuid.Nil) || (classID != "" && parseOptionalUUID(classID) == uuid.Nil) {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	if classID != "" && yearID == "" {
		http.Error(w, "Pilih tahun ajaran sebelum memilih kelas.", 400)
		return
	}
	if actor.Role == domain.RoleAdmin || (yearID != "" && (classID != "" || actor.Role == domain.RoleWaliKelas)) {
		out, err := h.students.List(r.Context(), service.ListStudentsInput{Actor: actor, IncludePlacement: true, AcademicYearID: parseOptionalUUID(yearID), ClassID: parseOptionalUUID(classID), UnassignedOnly: view == "unplaced", Options: service.ListOptions{Limit: limit + 1, Offset: (page - 1) * limit, IncludeTotal: true, Search: search, StudentStatus: status}})
		if err != nil {
			httperror.Write(w, err)
			return
		}
		appendStudentRows(&vm, out.Students)
		vm.Pagination = newPagination(r, page, out.Total, len(vm.Rows), vm.HasNext)
		for i := range vm.Rows {
			placement := out.Placements[parseOptionalUUID(vm.Rows[i].ID)]
			vm.Rows[i].AcademicYear, vm.Rows[i].Class = placement.AcademicYear, placement.Class
			if placement.AcademicYearID != uuid.Nil {
				vm.Rows[i].AcademicYearID = placement.AcademicYearID.String()
				if actor.Role == domain.RoleWaliKelas {
					vm.Rows[i].TransferURL = "/students/" + vm.Rows[i].ID + "/class-transfer-requests?academic_year_id=" + vm.Rows[i].AcademicYearID
				}
			}
		}
	}

	if redirectMissingPage(w, r, vm.Pagination) {
		return
	}
	h.renderStudent(w, vm)
}

func appendStudentRows(vm *StudentPageViewModel, students []*domain.Student) {
	for _, student := range students {
		if student != nil {
			vm.Rows = append(vm.Rows, StudentRow{ID: student.ID.String(), Name: student.Name, NIS: stringValue(student.NIS), NISN: stringValue(student.NISN), Status: interfaceStatusLabel(string(student.Status))})
		}
	}
	vm.HasNext = len(vm.Rows) > vm.Limit
	if vm.HasNext {
		vm.Rows = vm.Rows[:vm.Limit]
	}
	vm.HasPrevious, vm.PreviousPage, vm.NextPage = vm.Page > 1, vm.Page-1, vm.Page+1
}

func (h *StudentPageHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	if err := parseStudentForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	input, err := parseStudentMasterForm(r)
	if err != nil {
		h.formError(w, r, actor, http.StatusBadRequest, "NIS, NISN, dan nama wajib diisi.", "", input)
		return
	}
	yearID, classID := parseOptionalUUID(r.PostForm.Get("academic_year_id")), parseOptionalUUID(r.PostForm.Get("class_id"))
	if yearID == uuid.Nil || classID == uuid.Nil {
		h.formError(w, r, actor, http.StatusBadRequest, "Pilih tahun ajaran dan kelas.", "", input)
		return
	}
	_, err = h.students.Create(r.Context(), service.CreateStudentInput{Actor: actor, Name: input.Name, NIS: input.NIS, NISN: input.NISN, AcademicYearID: yearID, ClassID: classID})
	if err != nil {
		status := httperror.Status(err)
		if status >= 500 {
			httperror.Write(w, err)
			return
		}
		h.formError(w, r, actor, status, safeStudentError(err), "", input)
		return
	}
	http.Redirect(w, r, "/students?notice=created", http.StatusSeeOther)
}

func (h *StudentPageHandler) Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxStudentImportBytes)
	if err := r.ParseMultipartForm(maxStudentImportBytes); err != nil {
		h.renderImportIssues(w, r, actor, []service.StudentImportIssue{{Message: "Berkas tidak dapat dibaca atau melewati batas ukuran 5 MB."}})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		h.renderImportIssues(w, r, actor, []service.StudentImportIssue{{Message: "Pilih satu berkas Excel (.xlsx) untuk diimpor."}})
		return
	}
	defer file.Close()
	if header.Size > maxStudentImportBytes || strings.ToLower(filepath.Ext(header.Filename)) != ".xlsx" {
		h.renderImportIssues(w, r, actor, []service.StudentImportIssue{{Message: "Berkas harus berformat .xlsx dan berukuran maksimal 5 MB."}})
		return
	}
	rows, issues := parseStudentWorkbook(file)
	if len(issues) > 0 {
		h.renderImportIssues(w, r, actor, issues)
		return
	}
	output, err := h.students.Import(r.Context(), service.ImportStudentsInput{Actor: actor, Rows: rows})
	if err != nil {
		if httperror.Status(err) >= 500 {
			httperror.Write(w, err)
			return
		}
		h.renderImportIssues(w, r, actor, []service.StudentImportIssue{{Message: "Impor ditolak. Periksa kewenangan dan data berkas."}})
		return
	}
	if len(output.Issues) > 0 {
		h.renderImportIssues(w, r, actor, output.Issues)
		return
	}
	http.Redirect(w, r, "/students?notice=imported&count="+strconv.Itoa(len(output.Students)), http.StatusSeeOther)
}

func (h *StudentPageHandler) ImportTemplate(w http.ResponseWriter, r *http.Request) {
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
		httperror.Write(w, service.ErrForbidden)
		return
	}
	workbook := excelize.NewFile()
	defer workbook.Close()
	textStyle, err := workbook.NewStyle(&excelize.Style{NumFmt: 49})
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := workbook.SetColStyle("Sheet1", "A:E", textStyle); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	for column, heading := range []string{"name", "nis", "nisn", "academic_year", "class"} {
		cell, _ := excelize.CoordinatesToCellName(column+1, 1)
		if err := workbook.SetCellValue("Sheet1", cell, heading); err != nil {
			http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
	buffer, err := workbook.WriteToBuffer()
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="template-impor-siswa.xlsx"`)
	_, _ = w.Write(buffer.Bytes())
}

func parseStudentWorkbook(reader io.Reader) ([]service.StudentImportRow, []service.StudentImportIssue) {
	workbook, err := excelize.OpenReader(reader, excelize.Options{UnzipSizeLimit: maxStudentImportBytes, UnzipXMLSizeLimit: maxStudentImportBytes})
	if err != nil {
		return nil, []service.StudentImportIssue{{Message: "Berkas Excel rusak atau format .xlsx tidak valid."}}
	}
	defer workbook.Close()
	sheets := workbook.GetSheetList()
	if len(sheets) == 0 {
		return nil, []service.StudentImportIssue{{Message: "Berkas Excel tidak memiliki lembar kerja."}}
	}
	rows, err := workbook.GetRows(sheets[0])
	if err != nil || len(rows) < 2 {
		return nil, []service.StudentImportIssue{{Message: "Lembar pertama harus memuat header dan setidaknya satu baris siswa."}}
	}
	columns := make(map[string]int, len(rows[0]))
	for index, header := range rows[0] {
		name := strings.ToLower(strings.TrimSpace(header))
		if name == "" {
			continue
		}
		if _, exists := columns[name]; exists {
			return nil, []service.StudentImportIssue{{SpreadsheetRow: 1, Message: "Header kolom duplikat: " + name + "."}}
		}
		columns[name] = index
	}
	for _, required := range []string{"name", "nis", "nisn", "academic_year", "class"} {
		if _, ok := columns[required]; !ok {
			return nil, []service.StudentImportIssue{{SpreadsheetRow: 1, Message: "Header wajib tidak ditemukan: " + required + "."}}
		}
	}
	studentRows := make([]service.StudentImportRow, 0, len(rows)-1)
	for index, cells := range rows[1:] {
		rowNumber := index + 2
		if len(cells) == 0 || allBlank(cells) {
			continue
		}
		get := func(name string) string {
			column := columns[name]
			if column >= len(cells) {
				return ""
			}
			return strings.TrimSpace(cells[column])
		}
		studentRows = append(studentRows, service.StudentImportRow{SpreadsheetRow: rowNumber, Name: get("name"), NIS: get("nis"), NISN: get("nisn"), AcademicYear: get("academic_year"), Class: get("class")})
		if len(studentRows) > maxStudentImportRows {
			return nil, []service.StudentImportIssue{{SpreadsheetRow: rowNumber, Message: "Batas impor adalah 5.000 siswa per berkas."}}
		}
	}
	if len(studentRows) == 0 {
		return nil, []service.StudentImportIssue{{Message: "Tidak ditemukan baris siswa untuk diimpor."}}
	}
	return studentRows, nil
}

func allBlank(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func (h *StudentPageHandler) renderImportIssues(w http.ResponseWriter, r *http.Request, actor service.Actor, issues []service.StudentImportIssue) {
	vm := h.baseVM(actor, "Impor Siswa", "Berkas ditolak; tidak ada siswa yang disimpan.")
	vm.IsImport = true
	vm.ImportErrors = issues
	vm.Error = "Impor dibatalkan. Perbaiki seluruh kesalahan lalu unggah ulang berkas."
	if err := h.loadOptions(r, actor, &vm, "", false); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	h.renderStudent(w, vm)
}

func (h *StudentPageHandler) Detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	page, limit, pageErr := parsePageOptions(r)
	if pageErr != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	yearID := strings.TrimSpace(r.URL.Query().Get("academic_year_id"))
	if yearID != "" {
		parsedYearID, parseErr := uuid.Parse(yearID)
		if parseErr != nil || parsedYearID == uuid.Nil {
			http.Error(w, localizedStatusMessage(400), 400)
			return
		}
	}
	out, err := h.students.Get(r.Context(), service.GetStudentInput{Actor: actor, StudentID: id, AcademicYearID: parseOptionalUUID(yearID)})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	vm := h.baseVM(actor, "Detail Siswa", "Identitas dan riwayat kelas siswa.")
	vm.IsDetail = true
	vm.Page, vm.Limit = page, limit
	vm.StudentID = id.String()
	vm.Name = out.Student.Name
	vm.NIS = stringValue(out.Student.NIS)
	vm.NISN = stringValue(out.Student.NISN)
	vm.StudentStatus = interfaceStatusLabel(string(out.Student.Status))
	if r.URL.Query().Get("notice") == "left" {
		vm.Notice = "Status pindah sekolah dicatat. Tabungan belum diselesaikan; Wali Kelas perlu menyelesaikan tabungan dari menu Tabungan."
	}
	if out.Student.UserID != nil {
		vm.UserID = out.Student.UserID.String()
	}
	if err := h.loadOptions(r, actor, &vm, yearID, actor.Role == domain.RoleAdmin); err != nil {
		httperror.Write(w, err)
		return
	}
	if actor.Role == domain.RoleWaliKelas && yearID != "" {
		vm.TransferURL = "/students/" + id.String() + "/class-transfer-requests?academic_year_id=" + yearID
	}
	if yearID != "" {
		yid, parseErr := uuid.Parse(yearID)
		if parseErr != nil || yid == uuid.Nil {
			http.Error(w, localizedStatusMessage(400), 400)
			return
		}
		hist, historyErr := h.students.ListClassHistory(r.Context(), service.ListStudentClassHistoryInput{Actor: actor, StudentID: id, AcademicYearID: yid, Options: service.ListOptions{Limit: limit + 1, Offset: (page - 1) * limit, IncludeTotal: true}})
		if historyErr != nil {
			httperror.Write(w, historyErr)
			return
		}
		vm.AcademicYearID = yearID
		vm.HasNext = len(hist.Histories) > limit
		if vm.HasNext {
			hist.Histories = hist.Histories[:limit]
		}
		vm.Pagination = newPagination(r, page, hist.Total, len(hist.Histories), vm.HasNext)
		for _, item := range hist.Histories {
			if item == nil {
				continue
			}
			className := item.ClassID.String()
			for _, class := range vm.Classes {
				if class.ID == item.ClassID.String() {
					className = class.Name
					break
				}
			}
			to := "Berjalan"
			if item.Period.To != nil {
				to = item.Period.To.Format("2006-01-02")
			}
			vm.History = append(vm.History, StudentHistoryRow{AcademicYear: yearID, Class: className, From: item.Period.From.Format("2006-01-02"), To: to})
		}
	}
	if redirectMissingPage(w, r, vm.Pagination) {
		return
	}
	h.renderStudent(w, vm)
}

func (h *StudentPageHandler) Edit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	out, err := h.students.Get(r.Context(), service.GetStudentInput{Actor: actor, StudentID: id})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	vm := h.baseVM(actor, "Ubah Siswa", "Ubah data induk siswa.")
	vm.IsEdit = true
	vm.StudentID = id.String()
	vm.Name = out.Student.Name
	vm.NIS = stringValue(out.Student.NIS)
	vm.NISN = stringValue(out.Student.NISN)
	if out.Student.UserID != nil {
		vm.UserID = out.Student.UserID.String()
	}
	if err := h.loadOptions(r, actor, &vm, "", true); err != nil {
		httperror.Write(w, err)
		return
	}
	h.renderStudent(w, vm)
}

func (h *StudentPageHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	if err := parseStudentForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	input, err := parseStudentMasterForm(r)
	if err != nil {
		h.formError(w, r, actor, 400, "NIS, NISN, dan nama wajib diisi; user ID harus berupa UUID.", id.String(), input)
		return
	}
	_, err = h.students.Update(r.Context(), service.UpdateStudentInput{Actor: actor, StudentID: id, Name: input.Name, NIS: input.NIS, NISN: input.NISN, UserID: input.UserID})
	if err != nil {
		status := httperror.Status(err)
		if status >= 500 {
			httperror.Write(w, err)
			return
		}
		h.formError(w, r, actor, status, safeStudentError(err), id.String(), input)
		return
	}
	http.Redirect(w, r, "/students/"+id.String(), http.StatusSeeOther)
}

func (h *StudentPageHandler) AssignClass(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	if err := parseStudentForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	yearID, yerr := uuid.Parse(r.FormValue("academic_year_id"))
	classID, cerr := uuid.Parse(r.FormValue("class_id"))
	if err != nil || yerr != nil || cerr != nil || id == uuid.Nil || yearID == uuid.Nil || classID == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	_, err = h.students.AssignClass(r.Context(), service.AssignStudentClassInput{Actor: actor, StudentID: id, AcademicYearID: yearID, ClassID: classID})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/students/"+id.String()+"?academic_year_id="+yearID.String()+"&notice=left", http.StatusSeeOther)
}

func (h *StudentPageHandler) Leave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	if err := parseStudentForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	yearID, yerr := uuid.Parse(r.FormValue("academic_year_id"))
	date, derr := time.Parse("2006-01-02", strings.TrimSpace(r.FormValue("leaving_date")))
	if err != nil || yerr != nil || derr != nil || id == uuid.Nil || yearID == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	_, err = h.students.Leave(r.Context(), service.LeaveStudentInput{Actor: actor, StudentID: id, AcademicYearID: yearID, LeavingDate: date})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/students/"+id.String()+"?academic_year_id="+yearID.String(), http.StatusSeeOther)
}

func (h *StudentPageHandler) Reactivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	if err := parseStudentForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	yearID, yearErr := uuid.Parse(r.FormValue("academic_year_id"))
	classID, classErr := uuid.Parse(r.FormValue("class_id"))
	effectiveDate, dateErr := time.Parse("2006-01-02", strings.TrimSpace(r.FormValue("effective_date")))
	if err != nil || yearErr != nil || classErr != nil || dateErr != nil || id == uuid.Nil || yearID == uuid.Nil || classID == uuid.Nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	_, err = h.students.Reactivate(r.Context(), service.ReactivateStudentInput{
		Actor: actor, StudentID: id, AcademicYearID: yearID, ClassID: classID, EffectiveDate: effectiveDate,
	})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/students/"+id.String()+"?academic_year_id="+yearID.String(), http.StatusSeeOther)
}

type studentMasterForm struct {
	Name      string
	NIS, NISN *string
	UserID    *uuid.UUID
}

func parseStudentMasterForm(r *http.Request) (studentMasterForm, error) {
	name := strings.TrimSpace(r.FormValue("name"))
	nis := strings.TrimSpace(r.FormValue("nis"))
	nisn := strings.TrimSpace(r.FormValue("nisn"))
	if name == "" || nis == "" || nisn == "" {
		return studentMasterForm{Name: name, NIS: &nis, NISN: &nisn}, errors.New("required")
	}
	v := studentMasterForm{Name: name, NIS: &nis, NISN: &nisn}
	if raw := strings.TrimSpace(r.FormValue("user_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil || id == uuid.Nil {
			return v, errors.New("invalid user id")
		}
		v.UserID = &id
	}
	return v, nil
}
func parseStudentForm(w http.ResponseWriter, r *http.Request) error {
	contentType := strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0])
	if contentType != "application/x-www-form-urlencoded" {
		return errors.New("invalid content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxStudentFormBytes)
	return r.ParseForm()
}
func (h *StudentPageHandler) baseVM(actor service.Actor, title, description string) StudentPageViewModel {
	return StudentPageViewModel{Title: title, Heading: title, Description: description, Role: string(actor.Role), Username: actor.Username, ActiveNav: "students", IsAdmin: actor.Role == domain.RoleAdmin, Page: 1, Limit: defaultAcademicYearPageSize, ViewFilter: "all"}
}
func (h *StudentPageHandler) loadOptions(r *http.Request, actor service.Actor, vm *StudentPageViewModel, yearID string, withUsers bool) error {
	token, ok := middleware.CSRFToken(r)
	if !ok {
		return service.ErrInvalidDependency
	}
	vm.CSRFToken = token
	years, err := h.years.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: 100}})
	if err != nil {
		return err
	}
	for _, y := range years.AcademicYears {
		if y != nil {
			vm.Years = append(vm.Years, StudentOption{ID: y.ID.String(), Name: y.Name})
			if actor.Role == domain.RoleWaliKelas && yearID == "" && y.IsOpen() {
				yearID = y.ID.String()
				vm.AcademicYearID = yearID
			}
		}
	}
	if yearID != "" {
		id, err := uuid.Parse(yearID)
		if err != nil || id == uuid.Nil {
			return domain.ErrInvalidID
		}
		classes, err := h.classOptions(r, actor, id)
		if err != nil {
			return err
		}
		for _, class := range classes {
			vm.Classes = append(vm.Classes, StudentOption{ID: class.ID, Name: class.Name})
		}

	}
	if actor.Role == domain.RoleAdmin && !vm.IsDetail && !vm.IsEdit && !vm.IsImport {
		if vm.PlacementYearID == "" {
			vm.PlacementYearID = yearID
			if vm.PlacementYearID == "" {
				for _, year := range years.AcademicYears {
					if year != nil && year.IsOpen() {
						vm.PlacementYearID = year.ID.String()
						break
					}
				}
			}
		}
		if id := parseOptionalUUID(vm.PlacementYearID); id != uuid.Nil {
			classes, err := h.classOptions(r, actor, id)
			if err != nil {
				return err
			}
			vm.PlacementClasses = classes
		}
	}
	if withUsers && actor.Role == domain.RoleAdmin {
		users, err := h.users.ListByRole(r.Context(), service.ListUsersByRoleInput{Actor: actor, Role: domain.RoleSiswa, Options: service.ListOptions{Limit: 100}})
		if err != nil {
			return err
		}
		for _, u := range users.Users {
			if u != nil {
				vm.Users = append(vm.Users, StudentOption{ID: u.ID.String(), Name: u.Username})
			}
		}
	}
	return nil
}
func (h *StudentPageHandler) formError(w http.ResponseWriter, r *http.Request, actor service.Actor, status int, message, id string, input studentMasterForm) {
	vm := h.baseVM(actor, "Data Siswa", "Periksa kembali data yang dimasukkan.")
	vm.Error = message
	vm.Name = input.Name
	if input.NIS != nil {
		vm.NIS = *input.NIS
	}
	if input.NISN != nil {
		vm.NISN = *input.NISN
	}
	if input.UserID != nil {
		vm.UserID = input.UserID.String()
	}
	vm.StudentID = id
	vm.IsEdit = id != ""
	vm.PlacementYearID, vm.PlacementClassID = r.PostForm.Get("academic_year_id"), r.PostForm.Get("class_id")
	if err := h.loadOptions(r, actor, &vm, "", id != ""); err != nil {
		http.Error(w, localizedStatusMessage(500), 500)
		return
	}
	w.WriteHeader(status)
	_ = h.templates.ExecuteTemplate(w, "students", vm)
}
func (h *StudentPageHandler) renderStudent(w http.ResponseWriter, vm StudentPageViewModel) {
	if err := h.templates.ExecuteTemplate(w, "students", vm); err != nil {
		http.Error(w, localizedStatusMessage(500), 500)
	}
}

func limitStudentImportRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxStudentImportBytes)
		next.ServeHTTP(w, r)
	})
}
func parseOptionalUUID(raw string) uuid.UUID { id, _ := uuid.Parse(strings.TrimSpace(raw)); return id }
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func safeStudentError(err error) string {
	switch {
	case errors.Is(err, domain.ErrEmptyName):
		return "Nama siswa wajib diisi."
	case errors.Is(err, domain.ErrInvalidValue):
		return "Periksa data siswa dan kecocokan kelas dengan tahun ajaran."
	case errors.Is(err, domain.ErrInvalidDateRange):
		return "Tanggal efektif kelas harus berada di dalam tahun ajaran."
	case errors.Is(err, service.ErrForbidden):
		return "Anda tidak memiliki akses untuk tindakan ini."
	case errors.Is(err, domain.ErrInvalidTransition):
		return "Perubahan tidak dapat dilakukan pada status siswa saat ini."
	default:
		return "Data siswa tidak dapat disimpan. Pastikan NIS dan NISN belum digunakan."
	}
}

// StudentClassOption is also returned by the dependent class selector.
type StudentClassOption struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	EffectiveDate string `json:"effective_date"`
}

func (h *StudentPageHandler) classOptions(r *http.Request, actor service.Actor, yearID uuid.UUID) ([]StudentClassOption, error) {
	options := make([]StudentClassOption, 0)
	for offset := 0; ; offset += 250 {
		out, err := h.classes.List(r.Context(), service.ListClassesInput{Actor: actor, AcademicYearID: yearID, Options: service.ListOptions{Limit: 250, Offset: offset}})
		if err != nil {
			return nil, err
		}
		for _, class := range out.Classes {
			if class != nil && class.AcademicYearID == yearID && !class.EffectiveDate.IsZero() {
				options = append(options, StudentClassOption{ID: class.ID.String(), Name: class.Name, EffectiveDate: class.EffectiveDate.Format("2006-01-02")})
			}
		}
		if len(out.Classes) < 250 {
			break
		}
	}
	return options, nil
}

func (h *StudentPageHandler) ClassOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(401), 401)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	yearID := parseOptionalUUID(r.URL.Query().Get("academic_year_id"))
	if yearID == uuid.Nil {
		http.Error(w, localizedStatusMessage(400), 400)
		return
	}
	out, err := h.classOptions(r, actor, yearID)
	if err != nil {
		httperror.Write(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(out)
}
