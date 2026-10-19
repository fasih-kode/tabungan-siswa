package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestPaginationIntegrationCountsMatchFiltersAndAuthorizedScopes(t *testing.T) {
	f := newStudentPlacementFixture(t)
	ctx := context.Background()
	students := make([]domain.Student, 0, 23)
	for i := 0; i < 23; i++ {
		out, err := f.students.Create(ctx, f.createInput(fmt.Sprintf("%04d", 4000+i)))
		if err != nil {
			t.Fatal(err)
		}
		students = append(students, *out.Student)
	}
	for _, tc := range []struct {
		name                        string
		year                        uuid.UUID
		search                      string
		offset, wantTotal, wantRows int
	}{
		{"first page", f.year.ID, "", 0, 23, 11},
		{"last page", f.year.ID, "", 20, 23, 3},
		{"filtered search", f.year.ID, "Siswa 400", 0, 10, 10},
		{"another year", f.otherYear.ID, "", 0, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := f.students.List(ctx, service.ListStudentsInput{Actor: f.actor, AcademicYearID: tc.year, IncludePlacement: true, Options: service.ListOptions{IncludeTotal: true, Limit: 11, Offset: tc.offset, Search: tc.search, StudentStatus: "ACTIVE"}})
			if err != nil || out.Total != tc.wantTotal || len(out.Students) != tc.wantRows {
				t.Fatalf("total=%d rows=%d err=%v", out.Total, len(out.Students), err)
			}
		})
	}
	// Count must not remove the class scope from a teacher's view.
	teacher := newServiceScopeUser(t, domain.RoleWaliKelas)
	if err := f.repos.Users.Create(ctx, teacher); err != nil {
		t.Fatal(err)
	}
	period, err := domain.NewDateRange(f.year.StartDate, &f.year.EndDate)
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := domain.NewTeacherClassAssignment(teacher.ID, f.year.ID, f.class.ID, period)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.TeacherClassAssignments.Create(ctx, assignment); err != nil {
		t.Fatal(err)
	}
	classB, err := f.classes.Create(ctx, service.CreateClassInput{Actor: f.actor, Name: "VII B", Level: 7, AcademicYearID: f.year.ID, EffectiveDate: f.class.EffectiveDate})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		actor service.Actor
		total int
	}{{f.actor, 2}, {service.Actor{UserID: teacher.ID, Role: domain.RoleWaliKelas}, 1}} {
		out, err := f.classes.List(ctx, service.ListClassesInput{Actor: tc.actor, AcademicYearID: f.year.ID, Options: service.ListOptions{IncludeTotal: true, Limit: 11}})
		if err != nil || out.Total != tc.total || len(out.Classes) != tc.total {
			t.Fatalf("class total=%d rows=%d err=%v", out.Total, len(out.Classes), err)
		}
	}
	// The independent transfer filters use the same total as the paginated list.
	for _, student := range students[:12] {
		request, err := domain.NewClassTransferRequest(student.ID, f.year.ID, f.class.ID, classB.Class.ID, f.actor.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.repos.ClassTransferRequests.Create(ctx, request); err != nil {
			t.Fatal(err)
		}
	}
	transfers, err := service.NewClassTransferService(service.Dependencies{Repositories: f.repos, UOW: f.manager})
	if err != nil {
		t.Fatal(err)
	}
	pending, err := transfers.ListPending(ctx, service.ListPendingTransfersInput{Actor: f.actor, AcademicYearID: f.year.ID, Options: service.ListOptions{IncludeTotal: true, Limit: 11}})
	if err != nil || pending.Total != 12 || len(pending.TransferRequests) != 11 {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
	history, err := f.students.ListClassHistory(ctx, service.ListStudentClassHistoryInput{Actor: f.actor, StudentID: students[0].ID, AcademicYearID: f.year.ID, Options: service.ListOptions{IncludeTotal: true, Limit: 11}})
	if err != nil || history.Total != 1 || len(history.Histories) != 1 {
		t.Fatalf("history=%+v err=%v", history, err)
	}
	userTotal, err := f.repos.ListTotals.Users(ctx, domain.RoleWaliKelas)
	if err != nil || userTotal != 1 {
		t.Fatalf("users=%d err=%v", userTotal, err)
	}
	yearTotal, err := f.repos.ListTotals.AcademicYears(ctx)
	if err != nil || yearTotal != 2 {
		t.Fatalf("years=%d err=%v", yearTotal, err)
	}
	unplacedTotal, err := f.repos.ListTotals.StudentDirectory(ctx, repository.ListOptions{UnassignedOnly: true})
	if err != nil || unplacedTotal != 0 {
		t.Fatalf("unplaced=%d err=%v", unplacedTotal, err)
	}
}
