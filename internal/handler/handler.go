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
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/permhoot/pennypinch/internal/config"
	"github.com/permhoot/pennypinch/internal/service"
	"github.com/permhoot/pennypinch/internal/storage"
)

// Server wires together services and templates and exposes the HTTP routes.
type Server struct {
	svc    *service.Service
	cfg    *config.Manager
	table  pageTemplate
	month  pageTemplate
	year   pageTemplate
	static http.Handler
}

// New builds a Server with the given storage and config manager.
func New(store storage.Storage, cfg *config.Manager) (*Server, error) {
	s := &Server{
		svc:    &service.Service{Store: store, Config: cfg},
		cfg:    cfg,
		static: staticHandler(),
	}

	var err error
	if s.table, err = parsePage("layout", "partials", "table"); err != nil {
		return nil, err
	}
	if s.month, err = parsePage("layout", "partials", "monthly"); err != nil {
		return nil, err
	}
	if s.year, err = parsePage("layout", "partials", "yearly"); err != nil {
		return nil, err
	}
	return s, nil
}

// Router builds the chi router with all routes registered.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Handle("/static/*", http.StripPrefix("/static/", s.static))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Pages.
	r.Get("/", s.handleTablePage)
	r.Get("/monthly", s.handleMonthlyPage)
	r.Get("/yearly", s.handleYearlyPage)

	r.Route("/api", func(r chi.Router) {
		// Expenses (JSON).
		r.Get("/expenses", s.handleListExpenses)
		r.Post("/expenses", s.handleCreateExpense)
		r.Get("/expenses/export", s.handleExportCSV)
		r.Get("/expenses/table", s.handleTablePartial)
		r.Post("/expenses/batch-delete", s.handleBatchDelete)
		r.Get("/expenses/{id}", s.handleGetExpense)
		r.Put("/expenses/{id}", s.handleUpdateExpense)
		r.Delete("/expenses/{id}", s.handleDeleteExpense)

		// Config.
		r.Get("/config/categories", s.handleCategories)

		// Import.
		r.Post("/import/csv", s.handleImportCSV)
		r.Get("/import/template", s.handleImportTemplate)

		// Analytics.
		r.Get("/analytics/monthly/{year}/{month}", s.handleMonthly)
		r.Get("/analytics/yearly/{year}", s.handleYearly)
		r.Get("/analytics/summary", s.handleSummary)
	})

	return r
}

// writeJSON writes v as an indented JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// writeError writes a JSON error envelope.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeBadRequest(w http.ResponseWriter, err error) {
	writeError(w, http.StatusBadRequest, err.Error())
}

// parseDateParam parses an optional YYYY-MM-DD query parameter.
func parseDateParam(r *http.Request, key string) (*time.Time, error) {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// parseFloatParam parses an optional float query parameter.
func parseFloatParam(r *http.Request, key string) (*float64, error) {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// parseFilter builds a storage.Filter from query parameters.
func parseFilter(r *http.Request) (storage.Filter, error) {
	q := r.URL.Query()

	f := storage.Filter{
		Search:   strings.TrimSpace(q.Get("search")),
		Category: strings.TrimSpace(q.Get("category")),
		SortBy:   strings.TrimSpace(q.Get("sort")),
		SortDir:  strings.TrimSpace(q.Get("dir")),
		Page:     intParam(q.Get("page"), 1),
		PageSize: intParam(q.Get("page_size"), 50),
	}
	if f.PageSize < 1 {
		f.PageSize = 50
	}
	if f.PageSize > 200 {
		f.PageSize = 200
	}

	var err error
	if f.DateFrom, err = parseDateParam(r, "date_from"); err != nil {
		return f, err
	}
	if f.DateTo, err = parseDateParam(r, "date_to"); err != nil {
		return f, err
	}
	if f.AmountMin, err = parseFloatParam(r, "amount_min"); err != nil {
		return f, err
	}
	if f.AmountMax, err = parseFloatParam(r, "amount_max"); err != nil {
		return f, err
	}
	return f, nil
}

func intParam(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	return n
}

func staticHandler() http.Handler {
	sub, err := fs.Sub(webFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServerFS(sub)
}
