// Copyright © 2026 The Homeport Team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package handler

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/permhoot/pennypinch/internal/model"
	"github.com/permhoot/pennypinch/internal/storage"
)

type pageData struct {
	Theme            string
	Currency         string
	CurrencyPosition string
	CurrencyInCells  bool
	Active           string
}

type tablePageData struct {
	pageData
	Expenses []model.Expense
	Total    int64
	Pages    int
	Page     int
	Filter   storage.Filter
	Query    string
	Summary  model.Summary
}

type monthlyPageData struct {
	pageData
	Year      int
	Month     int
	PrevYear  int
	PrevMonth int
	NextYear  int
	NextMonth int
	Monthly   *model.MonthlyOverview
}

type yearlyPageData struct {
	pageData
	Year     int
	PrevYear int
	NextYear int
	Income   float64
	Expenses float64
	Net      float64
	Sankey   *model.SankeyData
}

type tablePartialData struct {
	Expenses         []model.Expense
	Total            int64
	Pages            int
	Page             int
	Query            string
	Currency         string
	CurrencyPosition string
	CurrencyInCells  bool
	Filter           storage.Filter
}

func (s *Server) handleTablePage(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.Snapshot()
	f, err := parseFilter(r)
	if err != nil {
		writeBadRequest(w, err)
		return
	}

	expenses, total, err := s.svc.Store.ListExpenses(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sum, err := s.svc.Store.Summary(r.Context())
	if err != nil {
		sum = model.Summary{}
	}

	pages := int(math.Ceil(float64(total) / float64(f.PageSize)))
	if pages < 1 {
		pages = 1
	}

	data := tablePageData{
		pageData: pageData{
			Theme:            cfg.Settings.Theme,
			Currency:         cfg.Settings.CurrencySymbol,
			CurrencyPosition: cfg.Settings.CurrencyPosition,
			CurrencyInCells:  cfg.Settings.CurrencyInCells,
			Active:           "table",
		},
		Expenses: expenses,
		Total:    total,
		Pages:    pages,
		Page:     f.Page,
		Filter:   f,
		Query:    filterQueryString(f, false),
		Summary:  sum,
	}
	s.table.render(w, "layout", data)
}

func (s *Server) handleTablePartial(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.Snapshot()
	f, err := parseFilter(r)
	if err != nil {
		writeBadRequest(w, err)
		return
	}

	expenses, total, err := s.svc.Store.ListExpenses(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pages := int(math.Ceil(float64(total) / float64(f.PageSize)))
	if pages < 1 {
		pages = 1
	}

	data := tablePartialData{
		Expenses:         expenses,
		Total:            total,
		Pages:            pages,
		Page:             f.Page,
		Query:            filterQueryString(f, false),
		Currency:         cfg.Settings.CurrencySymbol,
		CurrencyPosition: cfg.Settings.CurrencyPosition,
		CurrencyInCells:  cfg.Settings.CurrencyInCells,
		Filter:           f,
	}
	s.table.render(w, "table-body", data)
}

func (s *Server) handleMonthlyPage(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.Snapshot()

	year, month := currentYearMonth()
	if y := r.URL.Query().Get("year"); y != "" {
		year, _ = strconv.Atoi(y)
	}
	if m := r.URL.Query().Get("month"); m != "" {
		month, _ = strconv.Atoi(m)
	}
	if month < 1 || month > 12 {
		month = 1
	}

	ov, err := s.svc.MonthlyOverview(r.Context(), year, month)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	prevYear, prevMonth := shiftMonth(year, month, -1)
	nextYear, nextMonth := shiftMonth(year, month, 1)

	data := monthlyPageData{
		pageData: pageData{
			Theme:            cfg.Settings.Theme,
			Currency:         cfg.Settings.CurrencySymbol,
			CurrencyPosition: cfg.Settings.CurrencyPosition,
			CurrencyInCells:  cfg.Settings.CurrencyInCells,
			Active:           "monthly",
		},
		Year:      year,
		Month:     month,
		PrevYear:  prevYear,
		PrevMonth: prevMonth,
		NextYear:  nextYear,
		NextMonth: nextMonth,
		Monthly:   ov,
	}
	s.month.render(w, "layout", data)
}

func (s *Server) handleYearlyPage(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.Snapshot()

	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		year, _ = strconv.Atoi(y)
	}

	income, expense, err := s.svc.YearlySummary(r.Context(), year)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sankey, err := s.svc.YearlySankey(r.Context(), year)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := yearlyPageData{
		pageData: pageData{
			Theme:            cfg.Settings.Theme,
			Currency:         cfg.Settings.CurrencySymbol,
			CurrencyPosition: cfg.Settings.CurrencyPosition,
			CurrencyInCells:  cfg.Settings.CurrencyInCells,
			Active:           "yearly",
		},
		Year:     year,
		PrevYear: year - 1,
		NextYear: year + 1,
		Income:   income,
		Expenses: expense,
		Net:      income - expense,
		Sankey:   sankey,
	}
	s.year.render(w, "layout", data)
}

func shiftMonth(year, month, delta int) (int, int) {
	t := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, delta, 0)
	return t.Year(), int(t.Month())
}

func currentYearMonth() (int, int) {
	now := time.Now()
	return now.Year(), int(now.Month())
}

func filterQueryString(f storage.Filter, includePage bool) string {
	v := url.Values{}
	if f.Search != "" {
		v.Set("search", f.Search)
	}
	if f.Category != "" {
		v.Set("category", f.Category)
	}
	if f.SortBy != "" {
		v.Set("sort", f.SortBy)
	}
	if f.SortDir != "" {
		v.Set("dir", f.SortDir)
	}
	if f.PageSize != 0 && f.PageSize != 50 {
		v.Set("page_size", strconv.Itoa(f.PageSize))
	}
	if f.DateFrom != nil {
		v.Set("date_from", f.DateFrom.Format("2006-01-02"))
	}
	if f.DateTo != nil {
		v.Set("date_to", f.DateTo.Format("2006-01-02"))
	}
	if f.AmountMin != nil {
		v.Set("amount_min", fmt.Sprintf("%.2f", *f.AmountMin))
	}
	if f.AmountMax != nil {
		v.Set("amount_max", fmt.Sprintf("%.2f", *f.AmountMax))
	}
	if includePage && f.Page > 0 {
		v.Set("page", strconv.Itoa(f.Page))
	}
	return v.Encode()
}
