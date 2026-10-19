package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ ClassService = (*classService)(nil)

func (s *classService) Create(
	ctx context.Context,
	input CreateClassInput,
) (CreateClassOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return CreateClassOutput{}, err
	}

	class, err := domain.NewClass(input.Name, input.Level)
	if err != nil {
		return CreateClassOutput{}, err
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return CreateClassOutput{}, fmt.Errorf("begin create class transaction: %w", err)
	}

	ctx = txCtx
	committed := false
	defer func() {
		if ownsTransaction && !committed {
			_ = uow.Rollback()
		}
	}()

	if input.AcademicYearID == uuid.Nil {
		return CreateClassOutput{}, domain.ErrInvalidID
	}
	year, err := repos.AcademicYears.GetByIDForUpdate(ctx, input.AcademicYearID)
	if err != nil {
		return CreateClassOutput{}, err
	}
	if err := validateClassEffectiveDate(year, input.EffectiveDate); err != nil {
		return CreateClassOutput{}, err
	}
	class.AcademicYearID, class.EffectiveDate = year.ID, input.EffectiveDate
	exists, err := repos.Classes.ExistsByNameAndLevel(ctx, class.Name, class.Level, year.ID)
	if err != nil {
		return CreateClassOutput{}, err
	}
	if exists {
		return CreateClassOutput{}, repository.ErrConflict
	}

	if err := repos.Classes.Create(ctx, class); err != nil {
		return CreateClassOutput{}, err
	}

	if err := createClassAudit(ctx, repos, input.Actor, "CREATE", class, nil); err != nil {
		return CreateClassOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return CreateClassOutput{}, fmt.Errorf("commit create class transaction: %w", err)
		}
		committed = true
	}

	return CreateClassOutput{Class: &class}, nil
}

func (s *classService) Get(
	ctx context.Context,
	input GetClassInput,
) (GetClassOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return GetClassOutput{}, err
	}

	if input.ClassID == uuid.Nil {
		return GetClassOutput{}, domain.ErrInvalidID
	}

	class, err := s.deps.Repositories.Classes.GetByID(ctx, input.ClassID)
	if err != nil {
		return GetClassOutput{}, err
	}

	return GetClassOutput{Class: &class}, nil
}

func (s *classService) List(
	ctx context.Context,
	input ListClassesInput,
) (ListClassesOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListClassesOutput{}, err
	}

	if input.AcademicYearID == uuid.Nil {
		return ListClassesOutput{}, domain.ErrInvalidID
	}

	var countTeacherID, countStudentID uuid.UUID
	var classes []domain.Class
	var err error

	switch input.Actor.Role {
	case domain.RoleAdmin:
		classes, err = s.deps.Repositories.Classes.List(
			ctx,
			repository.ListOptions{
				Limit:          input.Options.Limit,
				Offset:         input.Options.Offset,
				AcademicYearID: input.AcademicYearID,
			},
		)

	case domain.RoleWaliKelas:
		countTeacherID = input.Actor.UserID
		classes, err = s.deps.Repositories.Classes.ListByTeacherAndAcademicYear(
			ctx,
			input.Actor.UserID,
			input.AcademicYearID,
			repository.ListOptions{
				Limit:  input.Options.Limit,
				Offset: input.Options.Offset,
			},
		)

	case domain.RoleSiswa:
		student, studentErr := s.deps.Repositories.Students.GetByUserID(
			ctx,
			input.Actor.UserID,
		)
		if studentErr != nil {
			return ListClassesOutput{}, studentErr
		}

		countStudentID = student.ID
		histories, historyErr := s.deps.Repositories.StudentClassHistories.
			ListByStudentAndAcademicYear(
				ctx,
				student.ID,
				input.AcademicYearID,
				repository.ListOptions{
					Limit:  input.Options.Limit,
					Offset: input.Options.Offset,
				},
			)
		if historyErr != nil {
			return ListClassesOutput{}, historyErr
		}

		classes = make([]domain.Class, 0, len(histories))
		seen := make(map[uuid.UUID]struct{}, len(histories))

		for _, history := range histories {
			if _, ok := seen[history.ClassID]; ok {
				continue
			}

			class, classErr := s.deps.Repositories.Classes.GetByID(
				ctx,
				history.ClassID,
			)
			if classErr != nil {
				return ListClassesOutput{}, classErr
			}

			seen[class.ID] = struct{}{}
			classes = append(classes, class)
		}

		sort.Slice(classes, func(i, j int) bool {
			if classes[i].Level != classes[j].Level {
				return classes[i].Level < classes[j].Level
			}
			if classes[i].Name != classes[j].Name {
				return classes[i].Name < classes[j].Name
			}
			return classes[i].ID.String() < classes[j].ID.String()
		})

	default:
		return ListClassesOutput{}, ErrForbidden
	}

	if err != nil {
		return ListClassesOutput{}, err
	}

	total := 0
	if input.Options.IncludeTotal {
		if s.deps.Repositories.ListTotals == nil {
			return ListClassesOutput{}, ErrInvalidDependency
		}
		total, err = s.deps.Repositories.ListTotals.Classes(ctx, input.AcademicYearID, countTeacherID, countStudentID)
		if err != nil {
			return ListClassesOutput{}, err
		}
	}
	return ListClassesOutput{Classes: classPointers(classes), Total: total}, nil
}

func (s *classService) Update(
	ctx context.Context,
	input UpdateClassInput,
) (UpdateClassOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return UpdateClassOutput{}, err
	}

	if input.ClassID == uuid.Nil {
		return UpdateClassOutput{}, domain.ErrInvalidID
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return UpdateClassOutput{}, fmt.Errorf("begin update class transaction: %w", err)
	}

	ctx = txCtx
	committed := false
	defer func() {
		if ownsTransaction && !committed {
			_ = uow.Rollback()
		}
	}()

	existing, err := repos.Classes.GetByID(ctx, input.ClassID)
	if err != nil {
		return UpdateClassOutput{}, err
	}

	updated, err := domain.NewClass(input.Name, input.Level)
	if err != nil {
		return UpdateClassOutput{}, err
	}

	updated.AcademicYearID = existing.AcademicYearID
	updated.EffectiveDate = existing.EffectiveDate
	if input.AcademicYearID != uuid.Nil {
		if existing.AcademicYearID != uuid.Nil && input.AcademicYearID != existing.AcademicYearID {
			return UpdateClassOutput{}, domain.ErrInvalidTransition
		}
		updated.AcademicYearID = input.AcademicYearID
	}
	if !input.EffectiveDate.IsZero() {
		updated.EffectiveDate = input.EffectiveDate
	}
	if updated.AcademicYearID != uuid.Nil {
		year, err := repos.AcademicYears.GetByIDForUpdate(ctx, updated.AcademicYearID)
		if err != nil {
			return UpdateClassOutput{}, err
		}
		if err := validateClassEffectiveDate(year, updated.EffectiveDate); err != nil {
			return UpdateClassOutput{}, err
		}
	}
	updated.ID = existing.ID
	updated.CreatedAt = existing.CreatedAt
	updated.UpdatedAt = time.Now()

	if updated.Name != existing.Name || updated.Level != existing.Level || updated.AcademicYearID != existing.AcademicYearID {
		exists, err := repos.Classes.ExistsByNameAndLevel(
			ctx,
			updated.Name,
			updated.Level,
			updated.AcademicYearID,
		)
		if err != nil {
			return UpdateClassOutput{}, err
		}
		if exists {
			return UpdateClassOutput{}, repository.ErrConflict
		}
	}

	if err := repos.Classes.Update(ctx, updated); err != nil {
		return UpdateClassOutput{}, err
	}

	if err := createClassAudit(
		ctx,
		repos,
		input.Actor,
		"UPDATE",
		updated,
		&existing,
	); err != nil {
		return UpdateClassOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return UpdateClassOutput{}, fmt.Errorf("commit update class transaction: %w", err)
		}
		committed = true
	}

	return UpdateClassOutput{Class: &updated}, nil
}

func createClassAudit(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	action string,
	class domain.Class,
	before *domain.Class,
) error {
	var beforeData []byte
	var err error

	if before != nil {
		beforeData, err = json.Marshal(before)
		if err != nil {
			return fmt.Errorf("marshal class audit before data: %w", err)
		}
	}

	afterData, err := json.Marshal(class)
	if err != nil {
		return fmt.Errorf("marshal class audit after data: %w", err)
	}

	_, err = recordAudit(ctx, repos, RecordAuditInput{
		Actor:      actor,
		Action:     action,
		EntityType: domain.AuditEntityClass,
		EntityID:   class.ID,
		Before:     beforeData,
		After:      afterData,
	})
	return err
}

func classPointers(classes []domain.Class) []*domain.Class {
	result := make([]*domain.Class, 0, len(classes))
	for i := range classes {
		class := classes[i]
		result = append(result, &class)
	}
	return result
}

func validateClassEffectiveDate(year domain.AcademicYear, date time.Time) error {
	if date.IsZero() || date.Before(year.StartDate) || !date.Before(year.EndDate) {
		return domain.ErrInvalidDateRange
	}
	return nil
}
