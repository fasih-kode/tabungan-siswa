package service_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestYearScopedClassMigrationPreservesHistoricalReferences(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	schema := "class_migration_" + uuid.New().String()[:8]
	if _, err = conn.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		_, _ = conn.ExecContext(ctx, "SET search_path TO public")
		if _, err := conn.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	if _, err = conn.ExecContext(ctx, "SET search_path TO "+schema+", public"); err != nil {
		t.Fatal(err)
	}
	apply := func(pattern string) {
		t.Helper()
		paths, err := filepath.Glob(filepath.Join("..", "..", "migrations", pattern))
		if err != nil || len(paths) != 1 {
			t.Fatalf("migration %s: paths=%v err=%v", pattern, paths, err)
		}
		sql, err := os.ReadFile(paths[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.ExecContext(ctx, string(sql)); err != nil {
			t.Fatalf("migration %s: %v", pattern, err)
		}
	}
	for i := 1; i <= 6; i++ {
		apply(fmt.Sprintf("%06d_*.up.sql", i))
	}
	// Two global classes are reused in two years, with student and teacher
	// histories plus transfer references in both years.
	_, err = conn.ExecContext(ctx, `
 INSERT INTO users(id,username,password_hash,role) VALUES ('10000000-0000-0000-0000-000000000001','teacher','hash','WALI_KELAS');
 INSERT INTO academic_years(id,name,start_date,end_date,status) VALUES
 ('20000000-0000-0000-0000-000000000001','2025/2026','2025-07-01','2026-07-01','OPEN');
 INSERT INTO classes(id,name,level) VALUES
 ('30000000-0000-0000-0000-000000000001','VII A',7),('30000000-0000-0000-0000-000000000002','VII B',7);
 INSERT INTO students(id,name,nis,nisn,status) VALUES ('40000000-0000-0000-0000-000000000001','Siswa','123','00123','ACTIVE');
 INSERT INTO student_class_histories(student_id,academic_year_id,class_id,validity) VALUES
 ('40000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','[2025-07-15,2026-07-01)');
 INSERT INTO teacher_class_assignments(user_id,academic_year_id,class_id,validity) VALUES
 ('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','[2025-07-01,2026-07-01)');
 INSERT INTO class_transfer_requests(student_id,academic_year_id,from_class_id,to_class_id,requested_by,status,reviewed_by,reviewed_at) VALUES
 ('40000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000001','APPROVED','10000000-0000-0000-0000-000000000001',NOW());
 UPDATE academic_years SET status='CLOSED';
 INSERT INTO academic_years(id,name,start_date,end_date,status) VALUES
 ('20000000-0000-0000-0000-000000000002','2026/2027','2026-07-01','2027-07-01','OPEN');
 INSERT INTO student_class_histories(student_id,academic_year_id,class_id,validity) VALUES
 ('40000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000001','[2026-07-20,2027-07-01)');
 INSERT INTO teacher_class_assignments(user_id,academic_year_id,class_id,validity) VALUES
 ('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000001','[2026-07-01,2027-07-01)');
 INSERT INTO class_transfer_requests(student_id,academic_year_id,from_class_id,to_class_id,requested_by,status) VALUES
 ('40000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000001','PENDING');
 `)
	if err != nil {
		t.Fatal(err)
	}
	apply("000007_*.up.sql")
	var classes, invalid int
	if err := conn.QueryRowContext(ctx, `SELECT count(*) FROM classes`).Scan(&classes); err != nil || classes != 4 {
		t.Fatalf("classes=%d err=%v", classes, err)
	}
	checks := []string{
		`SELECT count(*) FROM student_class_histories h JOIN classes c ON c.id=h.class_id WHERE c.academic_year_id<>h.academic_year_id OR c.name<>'VII A'`,
		`SELECT count(*) FROM teacher_class_assignments a JOIN classes c ON c.id=a.class_id WHERE c.academic_year_id<>a.academic_year_id OR c.name<>'VII A'`,
		`SELECT count(*) FROM class_transfer_requests r JOIN classes f ON f.id=r.from_class_id JOIN classes t ON t.id=r.to_class_id WHERE f.academic_year_id<>r.academic_year_id OR t.academic_year_id<>r.academic_year_id OR f.name<>'VII A' OR t.name<>'VII B'`,
		`SELECT count(*) FROM classes c JOIN academic_years y ON y.id=c.academic_year_id WHERE c.effective_date<>y.start_date`,
		`SELECT count(*) FROM student_class_histories WHERE (academic_year_id='20000000-0000-0000-0000-000000000001' AND validity<>'[2025-07-15,2026-07-01)'::daterange) OR (academic_year_id='20000000-0000-0000-0000-000000000002' AND validity<>'[2026-07-20,2027-07-01)'::daterange)`,
	}
	for _, query := range checks {
		if err := conn.QueryRowContext(ctx, query).Scan(&invalid); err != nil || invalid != 0 {
			t.Fatalf("invalid references=%d err=%v query=%s", invalid, err, query)
		}
	}
	// A configured cross-year reference is rejected by PostgreSQL as well.
	_, err = conn.ExecContext(ctx, `UPDATE student_class_histories SET class_id=(SELECT id FROM classes WHERE name='VII B' AND academic_year_id='20000000-0000-0000-0000-000000000001') WHERE academic_year_id='20000000-0000-0000-0000-000000000002'`)
	if err == nil {
		t.Fatal("database accepted a cross-year reference")
	}
	apply("000007_*.down.sql")
	if err := conn.QueryRowContext(ctx, `SELECT count(*) FROM classes`).Scan(&classes); err != nil || classes != 2 {
		t.Fatalf("restored global classes=%d err=%v", classes, err)
	}
	apply("000007_*.up.sql")
}
