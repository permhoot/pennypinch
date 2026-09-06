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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/permhoot/pennypinch/internal/model"
	"github.com/permhoot/pennypinch/internal/service"
)

type expenseListResponse struct {
	Expenses []model.Expense `json:"expenses"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Pages    int             `json:"pages"`
}

func (s *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, expenseListResponse{
		Expenses: expenses,
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
		Pages:    pages,
	})
}

func (s *Server) handleGetExpense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeBadRequest(w, fmt.Errorf("invalid id"))
		return
	}
	e, err := s.svc.Store.GetExpense(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "expense not found")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	var e model.Expense
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeBadRequest(w, fmt.Errorf("invalid JSON body: %w", err))
		return
	}
	if err := normalizeExpense(&e); err != nil {
		writeBadRequest(w, err)
		return
	}
	id, err := s.svc.Store.CreateExpense(r.Context(), &e)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	e.ID = id
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleUpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeBadRequest(w, fmt.Errorf("invalid id"))
		return
	}
	var e model.Expense
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeBadRequest(w, fmt.Errorf("invalid JSON body: %w", err))
		return
	}
	e.ID = id
	if err := normalizeExpense(&e); err != nil {
		writeBadRequest(w, err)
		return
	}
	if err := s.svc.Store.UpdateExpense(r.Context(), &e); err != nil {
		writeError(w, http.StatusNotFound, "expense not found")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) handleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeBadRequest(w, fmt.Errorf("invalid id"))
		return
	}
	if err := s.svc.Store.DeleteExpense(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "expense not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (s *Server) handleBatchDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadRequest(w, fmt.Errorf("invalid JSON body: %w", err))
		return
	}
	n, err := s.svc.Store.DeleteExpenses(r.Context(), body.IDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": n})
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{"categories": cfg.Categories})
}

func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	expenses, err := s.svc.Store.AllExpenses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="expenses.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write(service.CSVColumnNames)
	for _, e := range expenses {
		_ = cw.Write([]string{
			e.Date.String(),
			formatAmount(e.Amount),
			e.Subject,
			e.Description,
			e.Category,
		})
	}
	cw.Flush()
}

func (s *Server) handleImportCSV(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeBadRequest(w, fmt.Errorf("failed to read upload: %w", err))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeBadRequest(w, fmt.Errorf("missing file field: %w", err))
		return
	}
	defer func() { _ = file.Close() }()

	parsed, errs := service.ParseCSV(file)
	if len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, service.ImportResult{
			Imported: 0,
			Errors:   errs,
		})
		return
	}
	if len(parsed) == 0 {
		writeBadRequest(w, fmt.Errorf("CSV contained no data rows"))
		return
	}

	result, err := s.svc.ImportExpenses(r.Context(), parsed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleImportTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="expense-template.csv"`)
	_, _ = w.Write([]byte(service.CSVTemplate()))
}

func (s *Server) handleMonthly(w http.ResponseWriter, r *http.Request) {
	year, err1 := strconv.Atoi(chi.URLParam(r, "year"))
	month, err2 := strconv.Atoi(chi.URLParam(r, "month"))
	if err1 != nil || err2 != nil || month < 1 || month > 12 {
		writeBadRequest(w, fmt.Errorf("invalid year/month"))
		return
	}
	ov, err := s.svc.MonthlyOverview(r.Context(), year, month)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ov)
}

func (s *Server) handleYearly(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		writeBadRequest(w, fmt.Errorf("invalid year"))
		return
	}
	data, err := s.svc.YearlySankey(r.Context(), year)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	sum, err := s.svc.Store.Summary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// normalizeExpense validates and canonicalizes an expense before storage.
func normalizeExpense(e *model.Expense) error {
	cat, err := service.NormalizeCategoryPath(e.Category)
	if err != nil {
		return err
	}
	e.Category = cat
	if e.Date.IsZero() {
		return fmt.Errorf("date is required (YYYY-MM-DD)")
	}
	e.Amount = math.Round(e.Amount*100) / 100
	return nil
}

func formatAmount(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
