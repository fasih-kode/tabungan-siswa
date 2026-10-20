package service

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type studentPromotionService struct{ deps Dependencies }

func NewStudentPromotionService(deps Dependencies) (*studentPromotionService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}
	return &studentPromotionService{deps: deps}, nil
}

func (s *studentPromotionService) Preview(ctx context.Context, input PreviewStudentPromotionInput) (PreviewStudentPromotionOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return PreviewStudentPromotionOutput{}, err
	}
	if input.FromAcademicYearID == uuid.Nil || input.ToAcademicYearID == uuid.Nil || input.FromAcademicYearID == input.ToAcademicYearID {
		return PreviewStudentPromotionOutput{}, domain.ErrInvalidID
	}
	return previewStudentPromotion(ctx, s.deps.Repositories, input.FromAcademicYearID, input.ToAcademicYearID)
}

func previewStudentPromotion(ctx context.Context, repos repository.RepositorySet, fromID, toID uuid.UUID) (PreviewStudentPromotionOutput, error) {
	from, err := repos.AcademicYears.GetByID(ctx, fromID)
	if err != nil {
		return PreviewStudentPromotionOutput{}, err
	}
	to, err := repos.AcademicYears.GetByID(ctx, toID)
	if err != nil {
		return PreviewStudentPromotionOutput{}, err
	}
	if err := validatePromotionYears(from, to); err != nil {
		return PreviewStudentPromotionOutput{}, err
	}
	classes, err := repos.Classes.List(ctx, repository.ListOptions{AcademicYearID: fromID, Limit: 1000})
	if err != nil {
		return PreviewStudentPromotionOutput{}, err
	}
	maxLevel := 0
	fromClasses := make(map[uuid.UUID]domain.Class, len(classes))
	for _, c := range classes {
		if c.AcademicYearID != fromID {
			continue
		}
		fromClasses[c.ID] = c
		if c.Level > maxLevel {
			maxLevel = c.Level
		}
	}
	targetRows, err := repos.Classes.List(ctx, repository.ListOptions{AcademicYearID: toID, Limit: 1000})
	if err != nil {
		return PreviewStudentPromotionOutput{}, err
	}
	targets := make([]StudentPromotionClassOption, 0, len(targetRows))
	byLevel := make(map[int][]domain.Class)
	for _, c := range targetRows {
		if c.AcademicYearID != toID {
			continue
		}
		targets = append(targets, StudentPromotionClassOption{ID: c.ID.String(), Name: c.Name, Level: c.Level})
		byLevel[c.Level] = append(byLevel[c.Level], c)
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Level == targets[j].Level {
			return targets[i].Name < targets[j].Name
		}
		return targets[i].Level < targets[j].Level
	})
	students := make([]domain.Student, 0)
	for offset := 0; ; offset += 500 {
		page, e := repos.Students.List(ctx, repository.ListOptions{Limit: 500, Offset: offset})
		if e != nil {
			return PreviewStudentPromotionOutput{}, e
		}
		students = append(students, page...)
		if len(page) < 500 {
			break
		}
	}
	out := PreviewStudentPromotionOutput{FromAcademicYear: &from, ToAcademicYear: &to, TargetClasses: targets, MaxSourceLevel: maxLevel}
	for _, st := range students {
		var item StudentPromotionCandidate
		item.StudentID, item.StudentName, item.Status = st.ID.String(), st.Name, string(st.Status)
		histories, e := repos.StudentClassHistories.ListByStudentAndAcademicYear(ctx, st.ID, fromID, repository.ListOptions{Limit: 5000})
		if e != nil {
			return PreviewStudentPromotionOutput{}, e
		}
		if len(histories) == 0 {
			item.Status = "Belum ditempatkan"
			out.Excluded = append(out.Excluded, item)
			continue
		}
		h := histories[len(histories)-1]
		class, ok := fromClasses[h.ClassID]
		if !ok {
			item.Status += " (kelas asal tak tersedia)"
			out.Excluded = append(out.Excluded, item)
			continue
		}
		item.FromClassID, item.FromClassName, item.FromLevel = class.ID.String(), class.Name, class.Level
		if !st.IsActive() {
			out.Excluded = append(out.Excluded, item)
			continue
		}
		pending, e := repos.ClassTransferRequests.HasPendingByStudent(ctx, st.ID)
		if e != nil {
			return PreviewStudentPromotionOutput{}, e
		}
		if pending {
			item.Status = "Permintaan transfer kelas masih menunggu"
			out.Excluded = append(out.Excluded, item)
			continue
		}
		graduationCandidate := class.Level == maxLevel && maxLevel > 0 && len(byLevel[class.Level+1]) == 0
		if graduationCandidate {
			item.GraduationCandidate = true
			item.DefaultTargetID = "__graduate__"
		} else if next := byLevel[class.Level+1]; len(next) == 1 {
			item.DefaultTargetID = next[0].ID.String()
		} else if len(byLevel[class.Level+1]) == 0 && len(byLevel[class.Level]) == 1 {
			item.DefaultTargetID = byLevel[class.Level][0].ID.String()
		}
		if !item.GraduationCandidate && len(byLevel[class.Level]) == 0 && len(byLevel[class.Level+1]) == 0 {
			item.Status = "Tidak ada kelas tujuan yang sesuai"
			out.Excluded = append(out.Excluded, item)
			continue
		}
		out.Students = append(out.Students, item)
	}
	sort.Slice(out.Students, func(i, j int) bool {
		if out.Students[i].FromClassName == out.Students[j].FromClassName {
			return out.Students[i].StudentName < out.Students[j].StudentName
		}
		return out.Students[i].FromClassName < out.Students[j].FromClassName
	})
	sort.Slice(out.Excluded, func(i, j int) bool { return out.Excluded[i].StudentName < out.Excluded[j].StudentName })
	return out, nil
}

func validatePromotionYears(from, to domain.AcademicYear) error {
	if from.Status != domain.AcademicYearClosed {
		return domain.ErrInvalidTransition
	}
	if !to.IsOpen() || to.StartDate.Before(from.EndDate) {
		return domain.ErrInvalidTransition
	}
	return nil
}

func (s *studentPromotionService) Process(ctx context.Context, input ProcessStudentPromotionInput) (ProcessStudentPromotionOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ProcessStudentPromotionOutput{}, err
	}
	if input.FromAcademicYearID == uuid.Nil || input.ToAcademicYearID == uuid.Nil || input.FromAcademicYearID == input.ToAcademicYearID || len(input.Decisions) == 0 || len(input.Decisions) > 5000 {
		return ProcessStudentPromotionOutput{}, domain.ErrInvalidValue
	}
	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return ProcessStudentPromotionOutput{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()
	repos := uow.Repositories()
	from, err := repos.AcademicYears.GetByIDForUpdate(ctx, input.FromAcademicYearID)
	if err != nil {
		return ProcessStudentPromotionOutput{}, err
	}
	to, err := repos.AcademicYears.GetByIDForUpdate(ctx, input.ToAcademicYearID)
	if err != nil {
		return ProcessStudentPromotionOutput{}, err
	}
	if err := validatePromotionYears(from, to); err != nil {
		return ProcessStudentPromotionOutput{}, err
	}
	// Require a decision for every eligible active student so the batch cannot
	// silently leave part of a class behind.
	preview, err := previewStudentPromotion(ctx, repos, from.ID, to.ID)
	if err != nil {
		return ProcessStudentPromotionOutput{}, err
	}
	if len(input.Decisions) != len(preview.Students) {
		return ProcessStudentPromotionOutput{}, domain.ErrInvalidTransition
	}
	decisionByID := make(map[uuid.UUID]StudentPromotionDecision, len(input.Decisions))
	for _, d := range input.Decisions {
		if d.StudentID == uuid.Nil || d.FromClassID == uuid.Nil {
			return ProcessStudentPromotionOutput{}, domain.ErrInvalidID
		}
		if _, ok := decisionByID[d.StudentID]; ok {
			return ProcessStudentPromotionOutput{}, domain.ErrInvalidValue
		}
		decisionByID[d.StudentID] = d
	}
	fromClassesRaw, err := repos.Classes.List(ctx, repository.ListOptions{AcademicYearID: from.ID, Limit: 1000})
	if err != nil {
		return ProcessStudentPromotionOutput{}, err
	}
	fromClasses := make(map[uuid.UUID]domain.Class, len(fromClassesRaw))
	maxLevel := 0
	for _, c := range fromClassesRaw {
		if c.AcademicYearID != from.ID {
			continue
		}
		fromClasses[c.ID] = c
		if c.Level > maxLevel {
			maxLevel = c.Level
		}
	}
	result := ProcessStudentPromotionOutput{}
	for _, candidate := range preview.Students {
		id, _ := uuid.Parse(candidate.StudentID)
		d, ok := decisionByID[id]
		if !ok || d.FromClassID.String() != candidate.FromClassID {
			return ProcessStudentPromotionOutput{}, domain.ErrInvalidTransition
		}
		student, err := repos.Students.GetByIDForUpdate(ctx, id)
		if err != nil {
			return ProcessStudentPromotionOutput{}, err
		}
		if !student.IsActive() {
			return ProcessStudentPromotionOutput{}, domain.ErrInvalidTransition
		}
		pending, err := repos.ClassTransferRequests.HasPendingByStudent(ctx, id)
		if err != nil {
			return ProcessStudentPromotionOutput{}, err
		}
		if pending {
			return ProcessStudentPromotionOutput{}, domain.ErrInvalidTransition
		}
		if d.Graduate {
			if d.TargetClassID != uuid.Nil || !candidate.GraduationCandidate {
				return ProcessStudentPromotionOutput{}, domain.ErrInvalidValue
			}
			if err := student.Graduate(); err != nil {
				return ProcessStudentPromotionOutput{}, err
			}
			if err := repos.Students.Update(ctx, student); err != nil {
				return ProcessStudentPromotionOutput{}, err
			}
			if err := recordStudentPromotionAudit(ctx, repos, input.Actor, student.ID, from.ID, to.ID, d.FromClassID, uuid.Nil, true); err != nil {
				return ProcessStudentPromotionOutput{}, err
			}
			result.Graduated++
			continue
		}
		if d.TargetClassID == uuid.Nil {
			return ProcessStudentPromotionOutput{}, domain.ErrInvalidValue
		}
		target, err := repos.Classes.GetByID(ctx, d.TargetClassID)
		if err != nil {
			return ProcessStudentPromotionOutput{}, err
		}
		if target.AcademicYearID != to.ID {
			return ProcessStudentPromotionOutput{}, domain.ErrInvalidValue
		}
		source, ok := fromClasses[d.FromClassID]
		if !ok {
			return ProcessStudentPromotionOutput{}, domain.ErrInvalidTransition
		}
		if target.Level != source.Level && target.Level != source.Level+1 {
			return ProcessStudentPromotionOutput{}, domain.ErrInvalidValue
		}
		history, err := studentPlacementAtDate(ctx, repos, id, to.ID, target.ID, maxTime(to.StartDate, target.EffectiveDate))
		if err != nil {
			return ProcessStudentPromotionOutput{}, err
		}
		if err := repos.StudentClassHistories.Create(ctx, history); err != nil {
			return ProcessStudentPromotionOutput{}, err
		}
		if err := recordStudentPromotionAudit(ctx, repos, input.Actor, student.ID, from.ID, to.ID, d.FromClassID, target.ID, false); err != nil {
			return ProcessStudentPromotionOutput{}, err
		}
		result.Promoted++
	}
	if err := uow.Commit(); err != nil {
		return ProcessStudentPromotionOutput{}, err
	}
	committed = true
	return result, nil
}

func recordStudentPromotionAudit(ctx context.Context, repos repository.RepositorySet, actor Actor, studentID, fromYearID, toYearID, fromClassID, toClassID uuid.UUID, graduated bool) error {
	action, status := "STUDENT_PROMOTED", string(domain.StudentActive)
	if graduated {
		action, status = "STUDENT_GRADUATED", string(domain.StudentGraduated)
	}
	before, err := json.Marshal(map[string]any{"academic_year_id": fromYearID, "class_id": fromClassID, "status": string(domain.StudentActive)})
	if err != nil {
		return err
	}
	afterData := map[string]any{"academic_year_id": toYearID, "status": status}
	if !graduated {
		afterData["class_id"] = toClassID
	}
	after, err := json.Marshal(afterData)
	if err != nil {
		return err
	}
	_, err = recordAudit(ctx, repos, RecordAuditInput{Actor: actor, Action: action, EntityType: domain.AuditEntityStudent, EntityID: studentID, Before: before, After: after})
	return err
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
