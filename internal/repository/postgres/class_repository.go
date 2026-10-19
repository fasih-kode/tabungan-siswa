package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ repository.ClassRepository = (*ClassRepository)(nil)

type ClassRepository struct{ db database.DBTX }

func NewClassRepository(db database.DBTX) *ClassRepository { return &ClassRepository{db: db} }

func classConfigurationValues(c domain.Class) (any, any) {
	if c.AcademicYearID == uuid.Nil {
		return nil, nil
	}
	return c.AcademicYearID, c.EffectiveDate
}

func (r *ClassRepository) Create(ctx context.Context, c domain.Class) error {
	year, date := classConfigurationValues(c)
	_, err := r.db.ExecContext(ctx, `INSERT INTO classes
 (id, name, level, created_at, updated_at, academic_year_id, effective_date)
 VALUES ($1,$2,$3,$4,$5,$6,$7)`, c.ID, c.Name, c.Level, c.CreatedAt, c.UpdatedAt, year, date)
	if err != nil {
		return translateError("create class", err)
	}
	return nil
}

func scanClass(row interface{ Scan(...any) error }) (domain.Class, error) {
	var c domain.Class
	var year uuid.NullUUID
	var date sql.NullTime
	err := row.Scan(&c.ID, &c.Name, &c.Level, &c.CreatedAt, &c.UpdatedAt, &year, &date)
	if year.Valid {
		c.AcademicYearID = year.UUID
	}
	if date.Valid {
		c.EffectiveDate = date.Time
	}
	return c, err
}

const classColumns = `id, name, level, created_at, updated_at, academic_year_id, effective_date`

func (r *ClassRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Class, error) {
	c, err := scanClass(r.db.QueryRowContext(ctx, `SELECT `+classColumns+` FROM classes WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Class{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.Class{}, fmt.Errorf("get class by id: %w", err)
	}
	return c, nil
}

func readClasses(rows *sql.Rows) ([]domain.Class, error) {
	defer rows.Close()
	classes := make([]domain.Class, 0)
	for rows.Next() {
		c, err := scanClass(rows)
		if err != nil {
			return nil, fmt.Errorf("scan class: %w", err)
		}
		classes = append(classes, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate classes: %w", err)
	}
	return classes, nil
}

func (r *ClassRepository) List(ctx context.Context, opts repository.ListOptions) ([]domain.Class, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+classColumns+` FROM classes
 WHERE ($3::uuid = '00000000-0000-0000-0000-000000000000' OR academic_year_id=$3 OR academic_year_id IS NULL)
 ORDER BY level, name, id LIMIT $1 OFFSET $2`, opts.Limit, opts.Offset, opts.AcademicYearID)
	if err != nil {
		return nil, fmt.Errorf("list classes: %w", err)
	}
	return readClasses(rows)
}

func (r *ClassRepository) ListByTeacherAndAcademicYear(ctx context.Context, userID, yearID uuid.UUID, opts repository.ListOptions) ([]domain.Class, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+classColumns+` FROM classes c
 WHERE EXISTS (SELECT 1 FROM teacher_class_assignments a WHERE a.class_id=c.id AND a.user_id=$1 AND a.academic_year_id=$2)
 ORDER BY level, name, id LIMIT $3 OFFSET $4`, userID, yearID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list classes by teacher and academic year: %w", err)
	}
	return readClasses(rows)
}

func (r *ClassRepository) Update(ctx context.Context, c domain.Class) error {
	year, date := classConfigurationValues(c)
	result, err := r.db.ExecContext(ctx, `UPDATE classes SET name=$1, level=$2,
 updated_at=$3, academic_year_id=$5, effective_date=$6 WHERE id=$4`, c.Name, c.Level, c.UpdatedAt, c.ID, year, date)
	if err != nil {
		return translateError("update class", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated class rows affected: %w", err)
	}
	if n == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *ClassRepository) ExistsByNameAndLevel(ctx context.Context, name string, level int, yearIDs ...uuid.UUID) (bool, error) {
	var year any
	if len(yearIDs) > 0 {
		year = yearIDs[0]
	}
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM classes WHERE name=$1 AND level=$2 AND ($3::uuid IS NULL OR academic_year_id=$3))`, name, level, year).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check class existence: %w", err)
	}
	return exists, nil
}
