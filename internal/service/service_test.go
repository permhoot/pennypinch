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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/permhoot/pennypinch/internal/config"
	"github.com/permhoot/pennypinch/internal/model"
	"github.com/permhoot/pennypinch/internal/storage"
)

func newTestService() *Service {
	dir := GinkgoT().TempDir()
	store, err := storage.OpenSQLite(filepath.Join(dir, "expenses.db"))
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(store.Close)

	cfg, err := config.LoadManager(filepath.Join(dir, "config.yaml"))
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(cfg.Close)

	return &Service{Store: store, Config: cfg}
}

func mustExpense(date string, amount float64, subject, category string) model.Expense {
	d, err := model.ParseDate(date)
	Expect(err).NotTo(HaveOccurred())
	cat, err := NormalizeCategoryPath(category)
	Expect(err).NotTo(HaveOccurred())
	return model.Expense{Date: d, Amount: amount, Subject: subject, Category: cat}
}

var _ = Describe("ImportExpenses", func() {
	It("imports records and auto-adds categories to config", func() {
		s := newTestService()
		ctx := context.Background()

		parsed := []ParsedExpense{
			{Expense: mustExpense("2024-01-05", -42.50, "A", "Food/Groceries"), Line: 2},
			{Expense: mustExpense("2024-01-06", 2500.00, "B", "Income/Salary"), Line: 3},
		}

		result, err := s.ImportExpenses(ctx, parsed)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Imported).To(Equal(2))
		Expect(result.NewCategories).To(HaveLen(2))

		cfg := s.Config.Snapshot()
		names := map[string]bool{}
		for _, c := range cfg.Categories {
			names[c.Name] = true
		}
		Expect(names).To(HaveKey("food/groceries"))
		Expect(names).To(HaveKey("income/salary"))
	})
})

var _ = Describe("MonthlyOverview", func() {
	It("aggregates into top-level categories", func() {
		s := newTestService()
		ctx := context.Background()

		parsed := []ParsedExpense{
			{Expense: mustExpense("2024-03-01", -42.50, "A", "food/groceries")},
			{Expense: mustExpense("2024-03-02", -10.00, "B", "food/restaurants")},
			{Expense: mustExpense("2024-03-03", -30.00, "C", "transportation/fuel")},
			{Expense: mustExpense("2024-03-04", 2500.00, "D", "income/salary")},
		}
		_, err := s.ImportExpenses(ctx, parsed)
		Expect(err).NotTo(HaveOccurred())

		ov, err := s.MonthlyOverview(ctx, 2024, 3)
		Expect(err).NotTo(HaveOccurred())
		Expect(ov.Income).To(Equal(2500.00))
		Expect(ov.Expenses).To(Equal(82.50))
		Expect(ov.Net).To(Equal(2417.50))

		byName := map[string]float64{}
		for _, c := range ov.ExpenseCategories {
			byName[c.Category] = c.Total
		}
		Expect(byName["food"]).To(Equal(52.50))
		Expect(byName["transportation"]).To(Equal(30.00))
	})
})

var _ = Describe("YearlySankey", func() {
	It("maps income to expense categories and savings", func() {
		s := newTestService()
		ctx := context.Background()

		parsed := []ParsedExpense{
			{Expense: mustExpense("2024-01-05", 2500.00, "Acme", "income/salary")},
			{Expense: mustExpense("2024-01-06", -500.00, "Rent", "housing/rent")},
			{Expense: mustExpense("2024-02-06", -100.00, "Shell", "transportation/fuel")},
		}
		_, err := s.ImportExpenses(ctx, parsed)
		Expect(err).NotTo(HaveOccurred())

		data, err := s.YearlySankey(ctx, 2024)
		Expect(err).NotTo(HaveOccurred())
		Expect(data.Nodes).NotTo(BeEmpty())
		Expect(data.Links).NotTo(BeEmpty())

		var housing, savings float64
		for _, l := range data.Links {
			if l.To == "cat:housing" {
				housing = l.Value
			}
			if l.To == "savings" {
				savings = l.Value
			}
		}
		Expect(housing).To(Equal(500.00))
		Expect(savings).To(Equal(1900.00))
	})
})
