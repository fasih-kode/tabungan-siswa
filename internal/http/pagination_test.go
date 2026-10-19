package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"testing"
)

func TestPaginationWindowAndBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name           string
		page, total    int
		want           []int
		previous, next bool
	}{
		{"empty", 1, 0, []int{1}, false, false},
		{"exact ten", 1, 10, []int{1}, false, false},
		{"eleventh row", 1, 11, []int{1, 2}, false, true},
		{"reference first page", 1, 100, []int{1, 2, 3, 0, 8, 9, 10}, false, true},
		{"last page", 10, 100, []int{1, 2, 3, 0, 8, 9, 10}, true, false},
		{"middle", 50, 1000, []int{1, 2, 3, 0, 49, 50, 51, 0, 98, 99, 100}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/students", nil)
			p := newPagination(req, tc.page, tc.total, 0, false)
			var got []int
			active := 0
			for _, item := range p.Items {
				got = append(got, item.Number)
				if item.Current {
					active++
					if item.Number != tc.page {
						t.Fatal("wrong active page")
					}
				}
			}
			if !reflect.DeepEqual(got, tc.want) || active != 1 || (p.PreviousURL != "") != tc.previous || (p.NextURL != "") != tc.next {
				t.Fatalf("pagination=%+v numbers=%v want=%v", p, got, tc.want)
			}
		})
	}
	req := httptest.NewRequest(http.MethodGet, "/students", nil)
	if p := newPagination(req, 1, 1_000_000_000, 0, false); len(p.Items) > 11 {
		t.Fatalf("unbounded page window: %d", len(p.Items))
	}
}
func TestPaginationPreservesFiltersAndCanonicalizesEmptyPages(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/students?q=Ani+%26+Budi&status=ACTIVE&academic_year_id=year&class_id=class&view=class&page=5&limit=100&notice=created", nil)
	p := newPagination(req, 5, 12, 0, false)
	response := httptest.NewRecorder()
	if !redirectMissingPage(response, req, p) || response.Code != http.StatusSeeOther {
		t.Fatal("missing page was not redirected")
	}
	target, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"q": "Ani & Budi", "status": "ACTIVE", "academic_year_id": "year", "class_id": "class", "view": "class", "page": "2"} {
		if target.Query().Get(key) != want {
			t.Fatalf("%s=%q want %q", key, target.Query().Get(key), want)
		}
	}
	if target.Query().Has("limit") || target.Query().Has("notice") {
		t.Fatalf("legacy pagination size/success notice leaked: %s", target)
	}
}
func TestEveryTableUsesTenRowsEvenWithLegacyLimit(t *testing.T) {
	for _, query := range []string{"", "?limit=2", "?limit=100"} {
		req := httptest.NewRequest(http.MethodGet, "/students"+query, nil)
		page, limit, err := parsePageOptions(req)
		if err != nil || page != 1 || limit != 10 {
			t.Fatalf("query=%s page=%d limit=%d err=%v", query, page, limit, err)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/students?page="+strconv.FormatUint(^uint64(0)>>1, 10), nil)
	if _, _, err := parsePageOptions(req); err == nil {
		t.Fatal("overflowing page accepted")
	}
}
