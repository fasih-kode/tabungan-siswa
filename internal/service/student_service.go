package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ StudentService = (*studentService)(nil)

func (s *studentService) Create(
	ctx context.Context,
	input CreateStudentInput,
) (CreateStudentOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return CreateStudentOutput{}, err
	}

	student, err := domain.NewStudent(
		input.Name,
		input.UserID,
		input.NIS,
		input.NISN,
	)
	if err != nil {
		return CreateStudentOutput{}, err
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return CreateStudentOutput{}, fmt.Errorf(
			"begin create student transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()
	if err := validateStudentUserLink(ctx, uow.Repositories(), input.UserID); err != nil {
		return CreateStudentOutput{}, err
	}

	history, err := configuredStudentPlacement(ctx, uow.Repositories(), student.ID, input.AcademicYearID, input.ClassID)
	if err != nil {
		return CreateStudentOutput{}, err
	}
	if err := uow.Repositories().Students.Create(ctx, student); err != nil {
		return CreateStudentOutput{}, err
	}

	if err := uow.Repositories().StudentClassHistories.Create(ctx, history); err != nil {
		return CreateStudentOutput{}, err
	}
	if err := uow.Commit(); err != nil {
		return CreateStudentOutput{}, fmt.Errorf(
			"commit create student transaction: %w",
			err,
		)
	}

	committed = true

	return CreateStudentOutput{
		Student: &student,
	}, nil
}

func (s *studentService) Import(ctx context.Context, input ImportStudentsInput) (ImportStudentsOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ImportStudentsOutput{}, err
	}
	if len(input.Rows) == 0 || len(input.Rows) > 5000 {
		return ImportStudentsOutput{Issues: []StudentImportIssue{{Message: "Berkas harus berisi 1 sampai 5.000 baris siswa."}}}, nil
	}
	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return ImportStudentsOutput{}, fmt.Errorf("begin import students transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()
	repos := uow.Repositories()
	issues := make([]StudentImportIssue, 0)
	addIssue := func(i int, message string) {
		row := input.Rows[i].SpreadsheetRow
		if row < 1 {
			row = i + 2
		}
		issues = append(issues, StudentImportIssue{SpreadsheetRow: row, Message: message})
	}
	yearsByName := make(map[string][]domain.AcademicYear)
	for offset := 0; ; offset += 250 {
		years, err := repos.AcademicYears.List(ctx, repository.ListOptions{Limit: 250, Offset: offset})
		if err != nil {
			return ImportStudentsOutput{}, err
		}
		for _, year := range years {
			yearsByName[strings.TrimSpace(year.Name)] = append(yearsByName[strings.TrimSpace(year.Name)], year)
		}
		if len(years) < 250 {
			break
		}
	}
	rowYears := make([]uuid.UUID, len(input.Rows))
	referenced := make(map[uuid.UUID]bool)
	for i, row := range input.Rows {
		matches := yearsByName[strings.TrimSpace(row.AcademicYear)]
		if len(matches) != 1 {
			addIssue(i, "Tahun ajaran wajib diisi dengan nama yang tersedia dan tidak ambigu.")
			continue
		}
		rowYears[i] = matches[0].ID
		referenced[matches[0].ID] = true
	}
	// All imports acquire year locks in the same order; class configuration and
	// academic-year bounds cannot change between validation and persistence.
	ids := make([]uuid.UUID, 0, len(referenced))
	for id := range referenced {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	classesByYear := make(map[uuid.UUID]map[string][]domain.Class)
	for _, id := range ids {
		if _, err := repos.AcademicYears.GetByIDForUpdate(ctx, id); err != nil {
			return ImportStudentsOutput{}, err
		}
		classesByYear[id] = make(map[string][]domain.Class)
		for offset := 0; ; offset += 250 {
			classes, err := repos.Classes.List(ctx, repository.ListOptions{AcademicYearID: id, Limit: 250, Offset: offset})
			if err != nil {
				return ImportStudentsOutput{}, err
			}
			for _, class := range classes {
				if class.AcademicYearID == id {
					classesByYear[id][strings.TrimSpace(class.Name)] = append(classesByYear[id][strings.TrimSpace(class.Name)], class)
				}
			}
			if len(classes) < 250 {
				break
			}
		}
	}
	students := make([]domain.Student, len(input.Rows))
	histories := make([]domain.StudentClassHistory, len(input.Rows))
	seenNIS, seenNISN := map[string]int{}, map[string]int{}
	for i, row := range input.Rows {
		name, nis, nisn := strings.TrimSpace(row.Name), strings.TrimSpace(row.NIS), strings.TrimSpace(row.NISN)
		if name == "" || nis == "" || nisn == "" {
			addIssue(i, "Nama, NIS, dan NISN wajib diisi.")
		}
		rowNumber := row.SpreadsheetRow
		if rowNumber < 1 {
			rowNumber = i + 2
		}
		for _, identity := range []struct {
			value, label string
			seen         map[string]int
			lookup       func(context.Context, string) (domain.Student, error)
		}{
			{nis, "NIS", seenNIS, repos.Students.GetByNIS}, {nisn, "NISN", seenNISN, repos.Students.GetByNISN},
		} {
			if identity.value == "" {
				continue
			}
			if prior, ok := identity.seen[identity.value]; ok {
				addIssue(i, fmt.Sprintf("%s duplikat dengan baris %d.", identity.label, prior))
			} else {
				identity.seen[identity.value] = rowNumber
			}
			if _, err := identity.lookup(ctx, identity.value); err == nil {
				addIssue(i, identity.label+" sudah digunakan siswa lain.")
			} else if !errors.Is(err, repository.ErrNotFound) {
				return ImportStudentsOutput{}, err
			}
		}
		student, err := domain.NewStudent(name, nil, &nis, &nisn)
		if err == nil {
			students[i] = student
		}
		if rowYears[i] == uuid.Nil {
			continue
		}
		matches := classesByYear[rowYears[i]][strings.TrimSpace(row.Class)]
		if len(matches) != 1 {
			addIssue(i, "Kelas wajib diisi dan harus cocok dengan tahun ajaran; nama kelas harus tidak ambigu.")
			continue
		}
		if err != nil {
			continue
		}
		history, err := configuredStudentPlacement(ctx, repos, student.ID, rowYears[i], matches[0].ID)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidDateRange) || errors.Is(err, domain.ErrInvalidValue) || errors.Is(err, domain.ErrInvalidID) {
				addIssue(i, "Konfigurasi tanggal efektif kelas tidak valid.")
				continue
			}
			return ImportStudentsOutput{}, err
		}
		histories[i] = history
	}
	if len(issues) > 0 {
		sort.SliceStable(issues, func(i, j int) bool { return issues[i].SpreadsheetRow < issues[j].SpreadsheetRow })
		return ImportStudentsOutput{Issues: issues}, nil
	}
	for i, student := range students {
		if err := repos.Students.Create(ctx, student); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				addIssue(i, "NIS atau NISN baru saja digunakan siswa lain. Unggah ulang setelah memperbaiki data.")
				return ImportStudentsOutput{Issues: issues}, nil
			}
			return ImportStudentsOutput{}, err
		}
		if err := repos.StudentClassHistories.Create(ctx, histories[i]); err != nil {
			return ImportStudentsOutput{}, err
		}
	}
	if err := uow.Commit(); err != nil {
		return ImportStudentsOutput{}, fmt.Errorf("commit imported students: %w", err)
	}
	committed = true
	return ImportStudentsOutput{Students: studentPointers(students)}, nil
}

// configuredStudentPlacement uses the class date only for the new history;
// changing a class configuration never rewrites existing student histories.
func configuredStudentPlacement(ctx context.Context, repos repository.RepositorySet, studentID, yearID, classID uuid.UUID) (domain.StudentClassHistory, error) {
	if studentID == uuid.Nil || yearID == uuid.Nil || classID == uuid.Nil {
		return domain.StudentClassHistory{}, domain.ErrInvalidID
	}
	year, err := repos.AcademicYears.GetByIDForUpdate(ctx, yearID)
	if err != nil {
		return domain.StudentClassHistory{}, err
	}
	return configuredStudentPlacementInYear(ctx, repos, studentID, year, classID)
}

// configuredStudentPlacementInYear memakai tahun terkunci agar pemanggil tidak meminta lock tahun setelah siswa.
func configuredStudentPlacementInYear(ctx context.Context, repos repository.RepositorySet, studentID uuid.UUID, year domain.AcademicYear, classID uuid.UUID) (domain.StudentClassHistory, error) {
	class, err := repos.Classes.GetByID(ctx, classID)
	if err != nil {
		return domain.StudentClassHistory{}, err
	}
	if class.AcademicYearID != year.ID {
		return domain.StudentClassHistory{}, domain.ErrInvalidValue
	}
	if err := validateClassEffectiveDate(year, class.EffectiveDate); err != nil {
		return domain.StudentClassHistory{}, err
	}
	period, err := domain.NewDateRange(class.EffectiveDate, &year.EndDate)
	if err != nil {
		return domain.StudentClassHistory{}, err
	}
	return domain.NewStudentClassHistory(studentID, year.ID, classID, period)
}

func (s *studentService) Get(
	ctx context.Context,
	input GetStudentInput,
) (GetStudentOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetStudentOutput{}, err
	}

	if input.StudentID == uuid.Nil {
		return GetStudentOutput{}, ErrScopeViolation
	}

	student, err := s.deps.Repositories.Students.GetByID(
		ctx,
		input.StudentID,
	)
	if err != nil {
		return GetStudentOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetStudentOutput{
			Student: &student,
		}, nil

	case domain.RoleSiswa:
		if student.UserID == nil {
			return GetStudentOutput{}, ErrScopeViolation
		}

		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetStudentOutput{}, err
		}

		return GetStudentOutput{
			Student: &student,
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil {
			return GetStudentOutput{}, ErrScopeViolation
		}

		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			student.ID,
			input.AcademicYearID,
		); err != nil {
			return GetStudentOutput{}, err
		}

		return GetStudentOutput{
			Student: &student,
		}, nil

	default:
		return GetStudentOutput{}, ErrForbidden
	}
}

func (s *studentService) GetByNIS(
	ctx context.Context,
	input GetStudentByNISInput,
) (GetStudentByNISOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetStudentByNISOutput{}, err
	}

	if input.NIS == "" {
		return GetStudentByNISOutput{}, domain.ErrInvalidValue
	}

	student, err := s.deps.Repositories.Students.GetByNIS(
		ctx,
		input.NIS,
	)
	if err != nil {
		return GetStudentByNISOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetStudentByNISOutput{
			Student: &student,
		}, nil

	case domain.RoleSiswa:
		if student.UserID == nil {
			return GetStudentByNISOutput{}, ErrScopeViolation
		}

		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetStudentByNISOutput{}, err
		}

		return GetStudentByNISOutput{
			Student: &student,
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil {
			return GetStudentByNISOutput{}, ErrScopeViolation
		}

		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			student.ID,
			input.AcademicYearID,
		); err != nil {
			return GetStudentByNISOutput{}, err
		}

		return GetStudentByNISOutput{
			Student: &student,
		}, nil

	default:
		return GetStudentByNISOutput{}, ErrForbidden
	}
}

func (s *studentService) GetByNISN(
	ctx context.Context,
	input GetStudentByNISNInput,
) (GetStudentByNISNOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetStudentByNISNOutput{}, err
	}

	if input.NISN == "" {
		return GetStudentByNISNOutput{}, domain.ErrInvalidValue
	}

	student, err := s.deps.Repositories.Students.GetByNISN(
		ctx,
		input.NISN,
	)
	if err != nil {
		return GetStudentByNISNOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetStudentByNISNOutput{
			Student: &student,
		}, nil

	case domain.RoleSiswa:
		if student.UserID == nil {
			return GetStudentByNISNOutput{}, ErrScopeViolation
		}

		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetStudentByNISNOutput{}, err
		}

		return GetStudentByNISNOutput{
			Student: &student,
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil {
			return GetStudentByNISNOutput{}, ErrScopeViolation
		}

		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			student.ID,
			input.AcademicYearID,
		); err != nil {
			return GetStudentByNISNOutput{}, err
		}

		return GetStudentByNISNOutput{
			Student: &student,
		}, nil

	default:
		return GetStudentByNISNOutput{}, ErrForbidden
	}
}

func (s *studentService) GetByUserID(
	ctx context.Context,
	input GetStudentByUserIDInput,
) (GetStudentByUserIDOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetStudentByUserIDOutput{}, err
	}

	if input.UserID == uuid.Nil {
		return GetStudentByUserIDOutput{}, ErrScopeViolation
	}

	student, err := s.deps.Repositories.Students.GetByUserID(
		ctx,
		input.UserID,
	)
	if err != nil {
		return GetStudentByUserIDOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetStudentByUserIDOutput{
			Student: &student,
		}, nil

	case domain.RoleSiswa:
		if err := RequireSelf(input.Actor, input.UserID); err != nil {
			return GetStudentByUserIDOutput{}, err
		}

		return GetStudentByUserIDOutput{
			Student: &student,
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil {
			return GetStudentByUserIDOutput{}, ErrScopeViolation
		}

		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			student.ID,
			input.AcademicYearID,
		); err != nil {
			return GetStudentByUserIDOutput{}, err
		}

		return GetStudentByUserIDOutput{
			Student: &student,
		}, nil

	default:
		return GetStudentByUserIDOutput{}, ErrForbidden
	}
}

func (s *studentService) List(
	ctx context.Context,
	input ListStudentsInput,
) (ListStudentsOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListStudentsOutput{}, err
	}

	options := repository.ListOptions{
		Limit:          input.Options.Limit,
		Offset:         input.Options.Offset,
		Search:         input.Options.Search,
		StudentStatus:  input.Options.StudentStatus,
		UnassignedOnly: input.UnassignedOnly,
	}
	if options.StudentStatus != "" &&
		options.StudentStatus != string(domain.StudentActive) &&
		options.StudentStatus != string(domain.StudentLeft) &&
		options.StudentStatus != string(domain.StudentGraduated) {
		return ListStudentsOutput{}, domain.ErrInvalidValue
	}

	if input.IncludePlacement {
		if input.ClassID != uuid.Nil && input.AcademicYearID == uuid.Nil {
			return ListStudentsOutput{}, domain.ErrInvalidID
		}
		switch input.Actor.Role {
		case domain.RoleAdmin:
		case domain.RoleWaliKelas:
			if input.AcademicYearID == uuid.Nil || input.UnassignedOnly {
				return ListStudentsOutput{}, ErrScopeViolation
			}
			if input.ClassID != uuid.Nil {
				if err := RequireClassScope(ctx, s.deps.Repositories, input.Actor, input.ClassID, input.AcademicYearID); err != nil {
					return ListStudentsOutput{}, err
				}
			} else {
				options.TeacherID = input.Actor.UserID
			}
		default:
			return ListStudentsOutput{}, ErrForbidden
		}
		options.AcademicYearID, options.ClassID = input.AcademicYearID, input.ClassID
		entries, err := s.deps.Repositories.Students.ListDirectory(ctx, options)
		if err != nil {
			return ListStudentsOutput{}, err
		}
		total := 0
		if input.Options.IncludeTotal {
			if s.deps.Repositories.ListTotals == nil {
				return ListStudentsOutput{}, ErrInvalidDependency
			}
			total, err = s.deps.Repositories.ListTotals.StudentDirectory(ctx, options)
			if err != nil {
				return ListStudentsOutput{}, err
			}
		}
		out := ListStudentsOutput{Total: total, Students: make([]*domain.Student, 0, len(entries)), Placements: make(map[uuid.UUID]StudentPlacement, len(entries))}
		for _, entry := range entries {
			student := entry.Student
			out.Students = append(out.Students, &student)
			out.Placements[student.ID] = StudentPlacement{AcademicYear: entry.AcademicYear, Class: entry.Class, AcademicYearID: entry.AcademicYearID, ClassID: entry.ClassID}
		}
		return out, nil
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		if input.AcademicYearID == uuid.Nil && input.ClassID == uuid.Nil {
			students, err := s.deps.Repositories.Students.List(ctx, options)
			if err != nil {
				return ListStudentsOutput{}, err
			}
			return ListStudentsOutput{Students: studentPointers(students)}, nil
		}
		if input.AcademicYearID == uuid.Nil || input.ClassID == uuid.Nil || input.UnassignedOnly {
			return ListStudentsOutput{}, ErrScopeViolation
		}
		students, err := s.deps.Repositories.Students.
			ListByClassAndAcademicYear(
				ctx,
				input.ClassID,
				input.AcademicYearID,
				options,
			)
		if err != nil {
			return ListStudentsOutput{}, err
		}

		return ListStudentsOutput{
			Students: studentPointers(students),
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil || input.ClassID == uuid.Nil || input.UnassignedOnly {
			return ListStudentsOutput{}, ErrScopeViolation
		}
		if err := RequireClassScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			input.ClassID,
			input.AcademicYearID,
		); err != nil {
			return ListStudentsOutput{}, err
		}

		students, err := s.deps.Repositories.Students.
			ListByClassAndAcademicYear(
				ctx,
				input.ClassID,
				input.AcademicYearID,
				options,
			)
		if err != nil {
			return ListStudentsOutput{}, err
		}

		return ListStudentsOutput{
			Students: studentPointers(students),
		}, nil

	case domain.RoleSiswa:
		return ListStudentsOutput{}, ErrForbidden

	default:
		return ListStudentsOutput{}, ErrForbidden
	}
}

func (s *studentService) ListClassHistory(
	ctx context.Context,
	input ListStudentClassHistoryInput,
) (ListStudentClassHistoryOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListStudentClassHistoryOutput{}, err
	}
	if input.StudentID == uuid.Nil || input.AcademicYearID == uuid.Nil {
		return ListStudentClassHistoryOutput{}, ErrScopeViolation
	}
	switch input.Actor.Role {
	case domain.RoleAdmin:
	case domain.RoleWaliKelas:
		if err := RequireStudentScope(ctx, s.deps.Repositories, input.Actor, input.StudentID, input.AcademicYearID); err != nil {
			return ListStudentClassHistoryOutput{}, err
		}
	case domain.RoleSiswa:
		student, err := s.deps.Repositories.Students.GetByID(ctx, input.StudentID)
		if err != nil {
			return ListStudentClassHistoryOutput{}, err
		}
		if student.UserID == nil {
			return ListStudentClassHistoryOutput{}, ErrScopeViolation
		}
		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return ListStudentClassHistoryOutput{}, err
		}
	default:
		return ListStudentClassHistoryOutput{}, ErrForbidden
	}
	histories, err := s.deps.Repositories.StudentClassHistories.ListByStudentAndAcademicYear(
		ctx, input.StudentID, input.AcademicYearID,
		repository.ListOptions{Limit: input.Options.Limit, Offset: input.Options.Offset},
	)
	if err != nil {
		return ListStudentClassHistoryOutput{}, err
	}
	result := make([]*domain.StudentClassHistory, 0, len(histories))
	for i := range histories {
		history := histories[i]
		result = append(result, &history)
	}
	total := 0
	if input.Options.IncludeTotal {
		if s.deps.Repositories.ListTotals == nil {
			return ListStudentClassHistoryOutput{}, ErrInvalidDependency
		}
		total, err = s.deps.Repositories.ListTotals.StudentClassHistory(ctx, input.StudentID, input.AcademicYearID)
		if err != nil {
			return ListStudentClassHistoryOutput{}, err
		}
	}
	return ListStudentClassHistoryOutput{Histories: result, Total: total}, nil
}

func (s *studentService) AssignClass(
	ctx context.Context,
	input AssignStudentClassInput,
) (AssignStudentClassOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return AssignStudentClassOutput{}, err
	}
	if input.StudentID == uuid.Nil || input.AcademicYearID == uuid.Nil || input.ClassID == uuid.Nil {
		return AssignStudentClassOutput{}, domain.ErrInvalidID
	}
	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return AssignStudentClassOutput{}, fmt.Errorf("begin student class assignment: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()
	repos := uow.Repositories()
	year, err := repos.AcademicYears.GetByIDForUpdate(ctx, input.AcademicYearID)
	if err != nil {
		return AssignStudentClassOutput{}, err
	}
	student, err := repos.Students.GetByIDForUpdate(ctx, input.StudentID)
	if err != nil {
		return AssignStudentClassOutput{}, err
	}
	if !student.IsActive() {
		return AssignStudentClassOutput{}, domain.ErrInvalidTransition
	}
	history, err := configuredStudentPlacementInYear(ctx, repos, input.StudentID, year, input.ClassID)
	if err != nil {
		return AssignStudentClassOutput{}, err
	}
	if err := repos.StudentClassHistories.Create(ctx, history); err != nil {
		return AssignStudentClassOutput{}, err
	}
	if err := uow.Commit(); err != nil {
		return AssignStudentClassOutput{}, fmt.Errorf("commit student class assignment: %w", err)
	}
	committed = true
	return AssignStudentClassOutput{History: &history}, nil
}

func (s *studentService) Update(
	ctx context.Context,
	input UpdateStudentInput,
) (UpdateStudentOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return UpdateStudentOutput{}, err
	}

	if input.StudentID == uuid.Nil {
		return UpdateStudentOutput{}, ErrScopeViolation
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return UpdateStudentOutput{}, fmt.Errorf(
			"begin update student transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()
	if err := validateStudentUserLink(ctx, repos, input.UserID); err != nil {
		return UpdateStudentOutput{}, err
	}

	existing, err := repos.Students.GetByIDForUpdate(
		ctx,
		input.StudentID,
	)
	if err != nil {
		return UpdateStudentOutput{}, err
	}

	student := existing
	student.UserID = input.UserID
	student.NIS = input.NIS
	student.NISN = input.NISN
	student.Name = input.Name
	student.UpdatedAt = time.Now()

	if err := validateStudentUpdate(student); err != nil {
		return UpdateStudentOutput{}, err
	}

	if err := repos.Students.Update(ctx, student); err != nil {
		return UpdateStudentOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return UpdateStudentOutput{}, fmt.Errorf(
			"commit update student transaction: %w",
			err,
		)
	}

	committed = true

	return UpdateStudentOutput{
		Student: &student,
	}, nil
}

func (s *studentService) Leave(
	ctx context.Context,
	input LeaveStudentInput,
) (LeaveStudentOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return LeaveStudentOutput{}, err
	}

	if input.StudentID == uuid.Nil || input.AcademicYearID == uuid.Nil {
		return LeaveStudentOutput{}, ErrScopeViolation
	}
	if input.LeavingDate.IsZero() {
		return LeaveStudentOutput{}, domain.ErrInvalidDateRange
	}

	// Pengesahan status LEFT wajib melalui Finalize proses pindah dua tahap.
	return LeaveStudentOutput{}, domain.ErrInvalidTransition
}

func (s *studentService) Reactivate(
	ctx context.Context,
	input ReactivateStudentInput,
) (ReactivateStudentOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ReactivateStudentOutput{}, err
	}
	if input.StudentID == uuid.Nil || input.AcademicYearID == uuid.Nil || input.ClassID == uuid.Nil {
		return ReactivateStudentOutput{}, domain.ErrInvalidID
	}
	if input.EffectiveDate.IsZero() {
		return ReactivateStudentOutput{}, domain.ErrInvalidDateRange
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return ReactivateStudentOutput{}, fmt.Errorf("begin reactivate student transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()
	year, err := repos.AcademicYears.GetByIDForUpdate(
		ctx,
		input.AcademicYearID,
	)
	if err != nil {
		return ReactivateStudentOutput{}, err
	}
	student, err := repos.Students.GetByIDForUpdate(ctx, input.StudentID)
	if err != nil {
		return ReactivateStudentOutput{}, err
	}
	if err := student.Reactivate(); err != nil {
		return ReactivateStudentOutput{}, err
	}
	history, err := studentPlacementAtDateInYear(
		ctx,
		repos,
		input.StudentID,
		year,
		input.ClassID,
		input.EffectiveDate,
	)
	if err != nil {
		return ReactivateStudentOutput{}, err
	}
	if err := repos.StudentClassHistories.Create(ctx, history); err != nil {
		return ReactivateStudentOutput{}, err
	}
	if err := repos.Students.Update(ctx, student); err != nil {
		return ReactivateStudentOutput{}, err
	}
	if err := uow.Commit(); err != nil {
		return ReactivateStudentOutput{}, fmt.Errorf("commit reactivate student transaction: %w", err)
	}
	committed = true
	return ReactivateStudentOutput{Student: &student, History: &history}, nil
}

func studentPlacementAtDate(ctx context.Context, repos repository.RepositorySet, studentID, yearID, classID uuid.UUID, effectiveDate time.Time) (domain.StudentClassHistory, error) {
	if studentID == uuid.Nil || yearID == uuid.Nil || classID == uuid.Nil {
		return domain.StudentClassHistory{}, domain.ErrInvalidID
	}
	year, err := repos.AcademicYears.GetByIDForUpdate(ctx, yearID)
	if err != nil {
		return domain.StudentClassHistory{}, err
	}
	return studentPlacementAtDateInYear(
		ctx,
		repos,
		studentID,
		year,
		classID,
		effectiveDate,
	)
}

func studentPlacementAtDateInYear(
	ctx context.Context,
	repos repository.RepositorySet,
	studentID uuid.UUID,
	year domain.AcademicYear,
	classID uuid.UUID,
	effectiveDate time.Time,
) (domain.StudentClassHistory, error) {
	if studentID == uuid.Nil || year.ID == uuid.Nil || classID == uuid.Nil {
		return domain.StudentClassHistory{}, domain.ErrInvalidID
	}
	class, err := repos.Classes.GetByID(ctx, classID)
	if err != nil {
		return domain.StudentClassHistory{}, err
	}
	if class.AcademicYearID != year.ID {
		return domain.StudentClassHistory{}, domain.ErrInvalidValue
	}
	if err := validateClassEffectiveDate(year, class.EffectiveDate); err != nil {
		return domain.StudentClassHistory{}, err
	}
	if err := validateClassEffectiveDate(year, effectiveDate); err != nil {
		return domain.StudentClassHistory{}, err
	}
	if effectiveDate.Before(class.EffectiveDate) {
		return domain.StudentClassHistory{}, domain.ErrInvalidDateRange
	}
	period, err := domain.NewDateRange(effectiveDate, &year.EndDate)
	if err != nil {
		return domain.StudentClassHistory{}, err
	}
	return domain.NewStudentClassHistory(studentID, year.ID, classID, period)
}

func validateStudentUpdate(student domain.Student) error {
	if student.ID == uuid.Nil {
		return ErrScopeViolation
	}

	if student.NIS == nil {
		return domain.ErrInvalidValue
	}

	if student.NISN == nil {
		return domain.ErrInvalidValue
	}

	if student.Name == "" {
		return domain.ErrEmptyName
	}

	if !student.Status.IsValid() {
		return domain.ErrInvalidValue
	}

	return nil
}

func validateStudentUserLink(ctx context.Context, repos repository.RepositorySet, userID *uuid.UUID) error {
	if userID == nil {
		return nil
	}
	if *userID == uuid.Nil || repos.Users == nil {
		return ErrInvalidDependency
	}
	user, err := repos.Users.GetByID(ctx, *userID)
	if err != nil {
		return err
	}
	if user.Role != domain.RoleSiswa {
		return domain.ErrInvalidValue
	}
	return nil
}

func studentPointers(students []domain.Student) []*domain.Student {
	result := make([]*domain.Student, 0, len(students))

	for i := range students {
		student := students[i]
		result = append(result, &student)
	}

	return result
}
