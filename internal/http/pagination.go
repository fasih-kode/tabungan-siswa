package http

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
)

const tablePageSize = 10

type PaginationItem struct {
	Number            int
	URL               string
	Current, Ellipsis bool
}
type PaginationViewModel struct {
	Page, TotalPages, TotalRows int
	PreviousURL, NextURL        string
	Items                       []PaginationItem
}

// Pagination URLs keep every active query filter and always use ten rows.
func paginationURL(r *http.Request, page int) string {
	query := r.URL.Query()
	query.Set("page", strconv.Itoa(page))
	query.Del("limit")
	// A POST success notice should not be replayed by subsequent pagination.
	for _, key := range []string{"notice", "count", "created", "reset", "username"} {
		query.Del(key)
	}
	return (&url.URL{Path: r.URL.Path, RawQuery: query.Encode()}).String()
}
func newPagination(r *http.Request, page, total, rows int, hasNext bool) PaginationViewModel {
	// Empty totals from test doubles or concurrent changes must not hide rows
	// already returned by the list query.
	if rows > 0 && total < (page-1)*tablePageSize+rows {
		total = (page-1)*tablePageSize + rows
	}
	if hasNext && total <= page*tablePageSize {
		total = page*tablePageSize + 1
	}
	pages := total / tablePageSize
	if total%tablePageSize != 0 {
		pages++
	}
	if pages < 1 {
		pages = 1
	}
	if page < 1 {
		page = 1
	}
	p := PaginationViewModel{Page: page, TotalPages: pages, TotalRows: total}
	if page > 1 {
		previous := page - 1
		if previous > pages {
			previous = pages
		}
		p.PreviousURL = paginationURL(r, previous)
	}
	if page < pages {
		p.NextURL = paginationURL(r, page+1)
	}

	candidates := []int{1, 2, 3, pages - 2, pages - 1, pages, page - 1, page, page + 1}
	if pages <= 7 {
		candidates = nil
		for n := 1; n <= pages; n++ {
			candidates = append(candidates, n)
		}
	}
	sort.Ints(candidates)
	previous := 0
	for _, n := range candidates {
		if n < 1 || n > pages || n == previous {
			continue
		}
		if previous > 0 && n-previous == 2 {
			p.Items = append(p.Items, PaginationItem{Number: previous + 1, URL: paginationURL(r, previous+1)})
		} else if previous > 0 && n-previous > 2 {
			p.Items = append(p.Items, PaginationItem{Ellipsis: true})
		}
		p.Items = append(p.Items, PaginationItem{Number: n, URL: paginationURL(r, n), Current: n == page})
		previous = n
	}

	return p
}

func redirectMissingPage(w http.ResponseWriter, r *http.Request, p PaginationViewModel) bool {
	if p.TotalPages > 0 && p.Page > p.TotalPages {
		http.Redirect(w, r, paginationURL(r, p.TotalPages), http.StatusSeeOther)
		return true
	}
	return false
}
