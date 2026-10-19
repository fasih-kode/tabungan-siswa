package http

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type PromotionYearOption struct{ ID, Name, Status string }
type PromotionTargetOption struct {
	ID, Name string
	Selected bool
}
type PromotionCandidateRow struct {
	service.StudentPromotionCandidate
	Targets []PromotionTargetOption
}
type StudentPromotionPageViewModel struct {
	Title, Heading, Description, CSRFToken, Role, Username, ActiveNav string
	FromYearID, ToYearID, Success                                     string
	FromYearName, ToYearName                                          string
	Years                                                             []PromotionYearOption
	Candidates                                                        []PromotionCandidateRow
	Excluded                                                          []service.StudentPromotionCandidate
	PageError                                                         string
}
type StudentPromotionPageHandler struct {
	promotions service.StudentPromotionService
	years      service.AcademicYearService
	templates  *template.Template
}

func limitPromotionRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		next.ServeHTTP(w, r)
	})
}

func NewStudentPromotionPageHandler(promotions service.StudentPromotionService, years service.AcademicYearService, templates *template.Template) (*StudentPromotionPageHandler, error) {
	if promotions == nil || years == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &StudentPromotionPageHandler{promotions: promotions, years: years, templates: templates}, nil
}

func (h *StudentPromotionPageHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(401), 401)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	vm := StudentPromotionPageViewModel{Title: "Kenaikan Kelas", Heading: "Kenaikan Kelas dan Kelulusan", Description: "Tinjau penempatan siswa dari tahun ajaran asal ke tahun tujuan.", Role: string(actor.Role), Username: actor.Username, ActiveNav: "student-promotions", FromYearID: r.URL.Query().Get("from_academic_year_id"), ToYearID: r.URL.Query().Get("to_academic_year_id")}
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	}
	if out, err := h.years.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: 250}}); err == nil {
		for _, y := range out.AcademicYears {
			if y != nil {
				vm.Years = append(vm.Years, PromotionYearOption{ID: y.ID.String(), Name: y.Name, Status: string(y.Status)})
			}
		}
	} else {
		httperror.Write(w, err)
		return
	}
	if p, e := strconv.Atoi(r.URL.Query().Get("promoted")); e == nil && p >= 0 {
		g, _ := strconv.Atoi(r.URL.Query().Get("graduated"))
		vm.Success = strconv.Itoa(p) + " siswa ditempatkan di tahun tujuan, " + strconv.Itoa(g) + " siswa dinyatakan lulus."
	}
	if vm.FromYearID != "" && vm.ToYearID != "" {
		from, e1 := uuid.Parse(vm.FromYearID)
		to, e2 := uuid.Parse(vm.ToYearID)
		if e1 != nil || e2 != nil {
			http.Error(w, http.StatusText(400), 400)
			return
		}
		out, err := h.promotions.Preview(r.Context(), service.PreviewStudentPromotionInput{Actor: actor, FromAcademicYearID: from, ToAcademicYearID: to})
		if err != nil {
			vm.PageError = "Pastikan tahun asal berstatus penutupan/ditutup dan tahun tujuan terbuka dengan tanggal yang berurutan."
		} else {
			vm.FromYearName, vm.ToYearName = out.FromAcademicYear.Name, out.ToAcademicYear.Name
			vm.Excluded = out.Excluded
			for _, c := range out.Students {
				row := PromotionCandidateRow{StudentPromotionCandidate: c}
				for _, target := range out.TargetClasses {
					if target.Level == c.FromLevel || target.Level == c.FromLevel+1 {
						row.Targets = append(row.Targets, PromotionTargetOption{ID: target.ID, Name: target.Name + " (tingkat " + strconv.Itoa(target.Level) + ")", Selected: target.ID == c.DefaultTargetID})
					}
				}
				vm.Candidates = append(vm.Candidates, row)
			}
		}
	}
	if err := h.templates.ExecuteTemplate(w, "student-promotions", vm); err != nil {
		http.Error(w, http.StatusText(500), 500)
	}
}

func (h *StudentPromotionPageHandler) Process(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(401), 401)
		return
	}
	if actor.Role != domain.RoleAdmin {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(400), 400)
		return
	}
	from, e1 := uuid.Parse(strings.TrimSpace(r.PostForm.Get("from_academic_year_id")))
	to, e2 := uuid.Parse(strings.TrimSpace(r.PostForm.Get("to_academic_year_id")))
	students, fromClasses, choices := r.PostForm["student_id"], r.PostForm["from_class_id"], r.PostForm["decision"]
	if e1 != nil || e2 != nil || len(students) == 0 || len(students) != len(fromClasses) || len(students) != len(choices) || len(students) > 5000 {
		http.Error(w, http.StatusText(400), 400)
		return
	}
	decisions := make([]service.StudentPromotionDecision, 0, len(students))
	for i := range students {
		sid, se := uuid.Parse(students[i])
		fid, fe := uuid.Parse(fromClasses[i])
		if se != nil || fe != nil {
			http.Error(w, http.StatusText(400), 400)
			return
		}
		d := service.StudentPromotionDecision{StudentID: sid, FromClassID: fid}
		if choices[i] == "__graduate__" {
			d.Graduate = true
		} else {
			d.TargetClassID, e1 = uuid.Parse(choices[i])
			if e1 != nil {
				http.Error(w, http.StatusText(400), 400)
				return
			}
		}
		decisions = append(decisions, d)
	}
	out, err := h.promotions.Process(r.Context(), service.ProcessStudentPromotionInput{Actor: actor, FromAcademicYearID: from, ToAcademicYearID: to, Decisions: decisions})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	q := url.Values{}
	q.Set("from_academic_year_id", from.String())
	q.Set("to_academic_year_id", to.String())
	q.Set("promoted", strconv.Itoa(out.Promoted))
	q.Set("graduated", strconv.Itoa(out.Graduated))
	http.Redirect(w, r, "/student-promotions?"+q.Encode(), http.StatusSeeOther)
}
