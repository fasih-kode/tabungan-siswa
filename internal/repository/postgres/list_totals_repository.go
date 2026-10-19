package postgres

import (
	"context"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type ListTotalsRepository struct{ db database.DBTX }

var _ repository.ListTotalsRepository = (*ListTotalsRepository)(nil)

func NewListTotalsRepository(db database.DBTX) *ListTotalsRepository {
	return &ListTotalsRepository{db: db}
}
func (r *ListTotalsRepository) count(ctx context.Context, query string, args ...any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count list rows: %w", err)
	}
	return total, nil
}
func (r *ListTotalsRepository) AcademicYears(ctx context.Context) (int, error) {
	return r.count(ctx, `SELECT count(*) FROM academic_years`)
}
func (r *ListTotalsRepository) Users(ctx context.Context, role domain.UserRole) (int, error) {
	return r.count(ctx, `SELECT count(*) FROM users WHERE role=$1`, role)
}
func (r *ListTotalsRepository) Classes(ctx context.Context, yearID, teacherID, studentID uuid.UUID) (int, error) {
	return r.count(ctx, `SELECT count(*) FROM classes c
 WHERE ($2::uuid<>'00000000-0000-0000-0000-000000000000' OR $3::uuid<>'00000000-0000-0000-0000-000000000000' OR c.academic_year_id=$1 OR c.academic_year_id IS NULL)
 AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR EXISTS(SELECT 1 FROM teacher_class_assignments a WHERE a.class_id=c.id AND a.user_id=$2 AND a.academic_year_id=$1))
 AND ($3::uuid='00000000-0000-0000-0000-000000000000' OR EXISTS(SELECT 1 FROM student_class_histories h WHERE h.class_id=c.id AND h.student_id=$3 AND h.academic_year_id=$1))`, yearID, teacherID, studentID)
}
func (r *ListTotalsRepository) StudentDirectory(ctx context.Context, opts repository.ListOptions) (int, error) {
	return r.count(ctx, `SELECT count(*)`+studentDirectoryFromWhere, opts.AcademicYearID, opts.ClassID, opts.Search, opts.StudentStatus, opts.UnassignedOnly, opts.TeacherID)
}
func (r *ListTotalsRepository) StudentClassHistory(ctx context.Context, studentID, yearID uuid.UUID) (int, error) {
	return r.count(ctx, `SELECT count(*) FROM student_class_histories WHERE student_id=$1 AND academic_year_id=$2`, studentID, yearID)
}
func (r *ListTotalsRepository) Transfers(ctx context.Context, studentID, yearID uuid.UUID, pendingOnly bool) (int, error) {
	return r.count(ctx, `SELECT count(*) FROM class_transfer_requests WHERE academic_year_id=$2 AND ($1::uuid='00000000-0000-0000-0000-000000000000' OR student_id=$1) AND (NOT $3 OR status='PENDING')`, studentID, yearID, pendingOnly)
}
