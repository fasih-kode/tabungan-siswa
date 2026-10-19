package app

import (
	"html/template"
	"strings"
	"testing"

	httpapp "github.com/fasih/tabungan-siswa/internal/http"
	"golang.org/x/net/html"
)

func nodesMatching(root *html.Node, match func(*html.Node) bool) []*html.Node {
	var result []*html.Node
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if match(n) {
			result = append(result, n)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return result
}
func htmlAttr(n *html.Node, name string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == name {
			return attr.Val, true
		}
	}
	return "", false
}
func instantAncestor(n *html.Node) *html.Node {
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		if _, ok := htmlAttr(parent, "data-instant-region"); ok {
			return parent
		}
	}
	return nil
}

func TestAllExistingSearchAndFilterFormsRenderInstantRegions(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, page string
		view       any
	}{
		{"students admin", "students", httpapp.StudentPageViewModel{Role: "ADMIN", IsAdmin: true, Page: 1}},
		{"students teacher", "students", httpapp.StudentPageViewModel{Role: "WALI_KELAS", Page: 1}},
		{"student history", "students", httpapp.StudentPageViewModel{Role: "ADMIN", IsAdmin: true, IsDetail: true, StudentID: "student", AcademicYearID: "year", StudentStatus: "ACTIVE"}},
		{"classes", "classes", httpapp.ClassPageViewModel{Role: "ADMIN", AcademicYearID: "year", Page: 1}},
		{"teacher assignments", "teacher-assignments", httpapp.TeacherAssignmentPageViewModel{Role: "ADMIN", AcademicYearID: "year"}},
		{"users", "users", httpapp.UserPageViewModel{Role: "ADMIN", RoleFilter: "SISWA", Page: 1}},
		{"pending transfers", "class-transfers", httpapp.ClassTransferPageViewModel{Role: "ADMIN", AcademicYearID: "year", Page: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := templates.ExecuteTemplate(&out, tc.page, tc.view); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "/static/js/instant-search.js") {
				t.Fatal("instant search script is missing")
			}
			root, err := html.Parse(strings.NewReader(out.String()))
			if err != nil {
				t.Fatal(err)
			}
			forms := nodesMatching(root, func(n *html.Node) bool {
				method, _ := htmlAttr(n, "method")
				return n.Data == "form" && method == "get"
			})
			if len(forms) != 1 {
				t.Fatalf("GET filter forms=%d, want one", len(forms))
			}
			for _, form := range forms {
				if _, ok := htmlAttr(form, "data-instant-filter"); !ok {
					t.Fatal("manual GET filter remains")
				}
				region := instantAncestor(form)
				if region == nil {
					t.Fatal("filter has no update region")
				}
				targets := nodesMatching(region, func(n *html.Node) bool { _, ok := htmlAttr(n, "data-instant-results"); return ok })
				if len(targets) != 1 {
					t.Fatalf("result containers=%d, want one", len(targets))
				}
				for parent := form; parent != nil; parent = parent.Parent {
					if parent == targets[0] {
						t.Fatal("result replacement would remove the focused search field")
					}
				}
				status := nodesMatching(region, func(n *html.Node) bool { _, ok := htmlAttr(n, "data-instant-status"); return ok })
				if len(status) != 1 {
					t.Fatal("missing live loading/error status")
				}
				for _, button := range nodesMatching(form, func(n *html.Node) bool { return n.Data == "button" }) {
					fallback := false
					for parent := button.Parent; parent != nil && parent != form; parent = parent.Parent {
						if parent.Data == "noscript" {
							fallback = true
						}
					}
					if !fallback {
						t.Fatal("manual search button remains visible with JavaScript enabled")
					}
				}
			}
			if tc.name == "students admin" {
				region := instantAncestor(forms[0])
				if region.Data != "section" {
					t.Fatal("search toolbar is not on the table card")
				}
				toolbar, _ := htmlAttr(forms[0], "class")
				if toolbar != "directory-toolbar" {
					t.Fatal("student search toolbar layout missing")
				}
				search := nodesMatching(forms[0], func(n *html.Node) bool { kind, _ := htmlAttr(n, "type"); return n.Data == "input" && kind == "search" })
				if len(search) != 1 {
					t.Fatal("student instant search field is missing")
				}
			}
		})
	}
}

func TestEveryRenderedTableIncludesSharedPagination(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, page string
		view       any
	}{
		{"students", "students", httpapp.StudentPageViewModel{Role: "ADMIN", IsAdmin: true}},
		{"history", "students", httpapp.StudentPageViewModel{Role: "ADMIN", IsAdmin: true, IsDetail: true, AcademicYearID: "year"}},
		{"academic years", "academic-years", httpapp.AcademicYearPageViewModel{Role: "ADMIN"}},
		{"classes", "classes", httpapp.ClassPageViewModel{Role: "ADMIN", AcademicYearID: "year"}},
		{"assignments", "teacher-assignments", httpapp.TeacherAssignmentPageViewModel{Role: "ADMIN", AcademicYearID: "year"}},
		{"users", "users", httpapp.UserPageViewModel{Role: "ADMIN", RoleFilter: "SISWA"}},
		{"pending transfers", "class-transfers", httpapp.ClassTransferPageViewModel{Role: "ADMIN"}},
		{"student transfers", "class-transfers", httpapp.ClassTransferPageViewModel{Role: "ADMIN", IsStudentList: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := templates.ExecuteTemplate(&out, tc.page, tc.view); err != nil {
				t.Fatal(err)
			}
			root, err := html.Parse(strings.NewReader(out.String()))
			if err != nil {
				t.Fatal(err)
			}
			tables := nodesMatching(root, func(n *html.Node) bool { return n.Data == "table" })
			pagers := nodesMatching(root, func(n *html.Node) bool {
				label, _ := htmlAttr(n, "aria-label")
				return n.Data == "nav" && label == "Navigasi halaman tabel"
			})
			if len(tables) != 1 || len(pagers) != len(tables) {
				t.Fatalf("tables=%d pagers=%d", len(tables), len(pagers))
			}
			if !strings.Contains(out.String(), "Previous") || !strings.Contains(out.String(), "Next") || !strings.Contains(out.String(), `aria-current="page"`) {
				t.Fatal("pagination controls missing")
			}
			if instantAncestor(pagers[0]) == nil {
				t.Fatal("pagination is outside its instant update region")
			}
		})
	}
}
