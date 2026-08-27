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
	"sort"

	"github.com/permhoot/pennypinch/internal/color"
	"github.com/permhoot/pennypinch/internal/model"
)

// MonthlyOverview aggregates a month's expenses into top-level categories,
// split between income and expense.
func (s *Service) MonthlyOverview(ctx context.Context, year, month int) (*model.MonthlyOverview, error) {
	totals, _, err := s.Store.MonthlyTotals(ctx, year, month)
	if err != nil {
		return nil, err
	}
	return s.buildOverview(year, month, totals), nil
}

// YearlySummary returns total income and expense (absolute value) for a year.
func (s *Service) YearlySummary(ctx context.Context, year int) (income, expense float64, err error) {
	totals, _, err := s.Store.YearlyTotals(ctx, year)
	if err != nil {
		return 0, 0, err
	}
	for _, t := range totals {
		if t.Total > 0 {
			income += t.Total
		} else {
			expense += -t.Total
		}
	}
	return income, expense, nil
}

func (s *Service) buildOverview(year, month int, totals map[string]model.CategoryTotal) *model.MonthlyOverview {
	ov := &model.MonthlyOverview{Year: year, Month: month}

	expenseByTop := make(map[string]model.CategoryTotal)
	incomeByTop := make(map[string]model.CategoryTotal)

	for _, t := range totals {
		top := TopLevelCategory(t.Category)
		if t.Total < 0 {
			agg := expenseByTop[top]
			agg.Category = top
			agg.Total += -t.Total
			agg.Count += t.Count
			expenseByTop[top] = agg
		} else {
			agg := incomeByTop[top]
			agg.Category = top
			agg.Total += t.Total
			agg.Count += t.Count
			incomeByTop[top] = agg
		}
	}

	ov.ExpenseCategories = sortedTotals(expenseByTop)
	ov.IncomeCategories = sortedTotals(incomeByTop)
	for _, c := range ov.ExpenseCategories {
		ov.Expenses += c.Total
	}
	for _, c := range ov.IncomeCategories {
		ov.Income += c.Total
	}
	ov.Net = ov.Income - ov.Expenses

	for i := range ov.ExpenseCategories {
		if ov.Expenses > 0 {
			ov.ExpenseCategories[i].Percent = ov.ExpenseCategories[i].Total / ov.Expenses * 100
		}
		ov.ExpenseCategories[i].Color = s.topLevelColor(ov.ExpenseCategories[i].Category)
	}
	for i := range ov.IncomeCategories {
		if ov.Income > 0 {
			ov.IncomeCategories[i].Percent = ov.IncomeCategories[i].Total / ov.Income * 100
		}
		ov.IncomeCategories[i].Color = s.topLevelColor(ov.IncomeCategories[i].Category)
	}
	return ov
}

// topLevelColor resolves a display color for a top-level category, preferring
// an exact config entry, then any subcategory entry, then the palette.
func (s *Service) topLevelColor(top string) string {
	cfg := s.Config.Snapshot()
	fallback := ""
	for _, c := range cfg.Categories {
		if c.Color == "" {
			continue
		}
		if c.Name == top {
			return color.NormalizeColorOr(c.Color, color.PaletteColor(top, color.DefaultPalette))
		}
		if fallback == "" && TopLevelCategory(c.Name) == top {
			fallback = c.Color
		}
	}
	if fallback != "" {
		return color.NormalizeColorOr(fallback, color.PaletteColor(top, color.DefaultPalette))
	}
	return color.PaletteColor(top, color.DefaultPalette)
}

func sortedTotals(m map[string]model.CategoryTotal) []model.CategoryTotal {
	out := make([]model.CategoryTotal, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out
}

// YearlySankey builds a cash-flow sankey: income sources flow into a total
// income hub, which then flows out to top-level expense categories and any
// remaining savings (or in from a deficit source when spending exceeds income).
func (s *Service) YearlySankey(ctx context.Context, year int) (*model.SankeyData, error) {
	totals, _, err := s.Store.YearlyTotals(ctx, year)
	if err != nil {
		return nil, err
	}
	sources, err := s.Store.YearlyIncomeBySubject(ctx, year)
	if err != nil {
		return nil, err
	}

	data := &model.SankeyData{
		Nodes: []model.SankeyNode{},
		Links: []model.SankeyLink{},
	}

	// Income hub.
	data.Nodes = append(data.Nodes, model.SankeyNode{ID: "income", Label: "Income", Color: "#8dd3c7"})

	totalIncome := 0.0
	sourceNames := make([]string, 0, len(sources))
	for name := range sources {
		sourceNames = append(sourceNames, name)
	}
	sort.Strings(sourceNames)
	for _, name := range sourceNames {
		v := sources[name]
		totalIncome += v
		id := "src:" + name
		data.Nodes = append(data.Nodes, model.SankeyNode{ID: id, Label: name, Color: color.PaletteColor("src:"+name, color.DefaultPalette)})
		data.Links = append(data.Links, model.SankeyLink{From: id, To: "income", Value: round2(v)})
	}

	// Expense categories (top-level).
	expenseByTop := make(map[string]model.CategoryTotal)
	for _, t := range totals {
		if t.Total >= 0 {
			continue
		}
		top := TopLevelCategory(t.Category)
		agg := expenseByTop[top]
		agg.Category = top
		agg.Total += -t.Total
		agg.Count += t.Count
		expenseByTop[top] = agg
	}
	expenseCats := sortedTotals(expenseByTop)

	totalExpense := 0.0
	for _, c := range expenseCats {
		totalExpense += c.Total
		id := "cat:" + c.Category
		nodeColor := s.topLevelColor(c.Category)
		data.Nodes = append(data.Nodes, model.SankeyNode{ID: id, Label: c.Category, Color: nodeColor})
		data.Links = append(data.Links, model.SankeyLink{From: "income", To: id, Value: round2(c.Total)})
	}

	// Balance the flow.
	if totalIncome >= totalExpense {
		savings := totalIncome - totalExpense
		if savings > 0.005 {
			data.Nodes = append(data.Nodes, model.SankeyNode{ID: "savings", Label: "Savings", Color: "#b3de69"})
			data.Links = append(data.Links, model.SankeyLink{From: "income", To: "savings", Value: round2(savings)})
		}
	} else {
		deficit := totalExpense - totalIncome
		data.Nodes = append(data.Nodes, model.SankeyNode{ID: "deficit", Label: "Deficit", Color: "#fb8072"})
		data.Links = append(data.Links, model.SankeyLink{From: "deficit", To: "income", Value: round2(deficit)})
	}

	return data, nil
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
