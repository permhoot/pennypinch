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

package storage

import (
	"context"
	"time"

	"github.com/permhoot/pennypinch/internal/model"
)

// Filter describes list-query constraints for expenses.
type Filter struct {
	Search    string
	Category  string
	DateFrom  *time.Time
	DateTo    *time.Time
	AmountMin *float64
	AmountMax *float64
	SortBy    string
	SortDir   string
	Page      int
	PageSize  int
}

// Storage is the persistence interface.
type Storage interface {
	Close() error

	CreateExpense(ctx context.Context, e *model.Expense) (int64, error)
	GetExpense(ctx context.Context, id int64) (*model.Expense, error)
	UpdateExpense(ctx context.Context, e *model.Expense) error
	DeleteExpense(ctx context.Context, id int64) error
	DeleteExpenses(ctx context.Context, ids []int64) (int64, error)

	ListExpenses(ctx context.Context, f Filter) ([]model.Expense, int64, error)
	AllExpenses(ctx context.Context) ([]model.Expense, error)

	// ImportExpenses inserts all expenses in a single transaction.
	ImportExpenses(ctx context.Context, expenses []model.Expense) (int64, error)

	// FindDuplicates returns the indices (into expenses) of rows that already
	// exist in the database, matched by (date, amount, subject, description).
	FindDuplicates(ctx context.Context, expenses []model.Expense) (map[int]bool, error)

	DistinctCategories(ctx context.Context) ([]string, error)

	MonthlyTotals(ctx context.Context, year, month int) (map[string]model.CategoryTotal, int64, error)
	YearlyTotals(ctx context.Context, year int) (map[string]model.CategoryTotal, int64, error)
	YearlyIncomeBySubject(ctx context.Context, year int) (map[string]float64, error)
	Summary(ctx context.Context) (model.Summary, error)
}
