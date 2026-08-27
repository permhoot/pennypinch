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
package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/permhoot/pennypinch/internal/config"
	"github.com/permhoot/pennypinch/internal/model"
	"github.com/permhoot/pennypinch/internal/storage"
)

// Service ties together persistence and configuration for business operations.
type Service struct {
	Store  storage.Storage
	Config *config.Manager
}

// ImportResult reports the outcome of a CSV import.
type ImportResult struct {
	Imported      int           `json:"imported"`
	NewCategories []string      `json:"new_categories"`
	Errors        []ImportError `json:"errors"`
}

// ImportExpenses validates already-parsed rows, adds any new categories to the
// config (atomically), then inserts the expenses in a single transaction. If
// the database insert fails, the config is rolled back to its prior state.
func (s *Service) ImportExpenses(ctx context.Context, parsed []ParsedExpense) (*ImportResult, error) {
	result := &ImportResult{Imported: len(parsed)}

	// Collect unique normalized category paths in first-seen order.
	seen := make(map[string]bool)
	var categories []string
	for _, p := range parsed {
		if p.Expense.Category != "" && !seen[p.Expense.Category] {
			seen[p.Expense.Category] = true
			categories = append(categories, p.Expense.Category)
		}
	}

	// Snapshot config categories so we can roll back on DB failure.
	before := s.Config.Snapshot().Categories

	newCats, err := s.Config.EnsureCategories(categories)
	if err != nil {
		return nil, fmt.Errorf("update config: %w", err)
	}
	for _, c := range newCats {
		result.NewCategories = append(result.NewCategories, c.Name)
	}

	expenses := make([]model.Expense, len(parsed))
	for i, p := range parsed {
		expenses[i] = p.Expense
	}

	if _, err := s.Store.ImportExpenses(ctx, expenses); err != nil {
		// Roll back the config additions.
		s.Config.RestoreCategories(before)
		return nil, fmt.Errorf("import to database: %w", err)
	}

	return result, nil
}

// DistinctCategories returns all category paths currently present in expenses.
func (s *Service) DistinctCategories(ctx context.Context) ([]string, error) {
	cats, err := s.Store.DistinctCategories(ctx)
	if err != nil {
		return nil, err
	}
	sort.Strings(cats)
	return cats, nil
}
