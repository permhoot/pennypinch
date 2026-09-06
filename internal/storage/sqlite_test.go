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
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/permhoot/pennypinch/internal/model"
)

var _ = Describe("SQLite", func() {
	var (
		s   *SQLite
		ctx context.Context
	)

	BeforeEach(func() {
		var err error
		s, err = OpenSQLite(filepath.Join(GinkgoT().TempDir(), "expenses.db"))
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
	})

	AfterEach(func() {
		Expect(s.Close()).To(Succeed())
	})

	insertExpense := func(date string, amount float64, subject, category string) int64 {
		d, err := model.ParseDate(date)
		Expect(err).NotTo(HaveOccurred())
		id, err := s.CreateExpense(ctx, &model.Expense{
			Date:     d,
			Amount:   amount,
			Subject:  subject,
			Category: category,
		})
		Expect(err).NotTo(HaveOccurred())
		return id
	}

	Describe("CRUD", func() {
		It("creates, reads, updates, and deletes an expense", func() {
			id := insertExpense("2024-01-05", -42.50, "Grocery Mart", "food/groceries")

			e, err := s.GetExpense(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(e.Category).To(Equal("food/groceries"))
			Expect(e.Amount).To(Equal(-42.50))
			Expect(e.Date.String()).To(Equal("2024-01-05"))

			e.Amount = -50.00
			Expect(s.UpdateExpense(ctx, e)).To(Succeed())
			updated, err := s.GetExpense(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Amount).To(Equal(-50.00))

			Expect(s.DeleteExpense(ctx, id)).To(Succeed())
			_, err = s.GetExpense(ctx, id)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("MonthlyTotals", func() {
		It("returns per-category totals for a single month", func() {
			insertExpense("2024-01-05", -42.50, "A", "food/groceries")
			insertExpense("2024-01-06", -10.00, "B", "food/restaurants")
			insertExpense("2024-01-07", -30.00, "C", "transportation/fuel")
			insertExpense("2024-01-08", 2500.00, "D", "income/salary")
			insertExpense("2024-02-01", -5.00, "E", "food/groceries")

			totals, count, err := s.MonthlyTotals(ctx, 2024, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(4)))
			Expect(totals["food/groceries"].Total).To(Equal(-42.50))
			Expect(totals["income/salary"].Total).To(Equal(2500.00))
		})
	})

	Describe("YearlyIncomeBySubject", func() {
		It("groups income by subject for a year", func() {
			insertExpense("2024-01-05", 2500.00, "Acme", "income/salary")
			insertExpense("2024-02-05", 300.00, "Freelance", "income/freelance")
			insertExpense("2024-03-05", 200.00, "", "income/other")
			insertExpense("2023-01-05", 999.00, "Old", "income/salary")

			bySubject, err := s.YearlyIncomeBySubject(ctx, 2024)
			Expect(err).NotTo(HaveOccurred())
			Expect(bySubject["Acme"]).To(Equal(2500.00))
			Expect(bySubject["other"]).To(Equal(200.00))
			Expect(bySubject).NotTo(HaveKey("Old"))
		})
	})

	Describe("Summary", func() {
		It("computes income, expense, net, and count", func() {
			insertExpense("2024-01-05", -42.50, "A", "food")
			insertExpense("2024-01-06", 100.00, "B", "income")

			sum, err := s.Summary(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(sum.TotalIncome).To(Equal(100.00))
			Expect(sum.TotalExpense).To(Equal(42.50))
			Expect(sum.Net).To(Equal(57.50))
			Expect(sum.Count).To(Equal(2))
		})
	})

	Describe("ListExpenses", func() {
		It("filters by search and category", func() {
			insertExpense("2024-01-05", -42.50, "Grocery Mart", "food/groceries")
			insertExpense("2024-01-06", -120.00, "Shell", "transportation/fuel")
			insertExpense("2024-02-01", 2500.00, "Acme", "income/salary")

			list, total, err := s.ListExpenses(ctx, Filter{Search: "shell"})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(list).To(HaveLen(1))
			Expect(list[0].Subject).To(Equal("Shell"))

			list, total, err = s.ListExpenses(ctx, Filter{Category: "food"})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(list[0].Category).To(Equal("food/groceries"))
		})
	})

	Describe("ImportExpenses", func() {
		It("inserts multiple rows in a transaction", func() {
			d, err := model.ParseDate("2024-01-05")
			Expect(err).NotTo(HaveOccurred())

			n, err := s.ImportExpenses(ctx, []model.Expense{
				{Date: d, Amount: -1.00, Subject: "a", Category: "x"},
				{Date: d, Amount: -2.00, Subject: "b", Category: "y"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(2)))

			_, total, err := s.ListExpenses(ctx, Filter{PageSize: 100})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
		})
	})

	Describe("FindDuplicates", func() {
		It("returns empty when no expenses exist", func() {
			d, err := model.ParseDate("2024-01-05")
			Expect(err).NotTo(HaveOccurred())

			dups, err := s.FindDuplicates(ctx, []model.Expense{
				{Date: d, Amount: -1.00, Subject: "a", Description: "desc"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(dups).To(BeEmpty())
		})

		It("detects exact duplicates", func() {
			d1, _ := model.ParseDate("2024-01-05")
			d2, _ := model.ParseDate("2024-01-06")
			insertExpense("2024-01-05", -42.50, "A", "")

			dups, err := s.FindDuplicates(ctx, []model.Expense{
				{Date: d1, Amount: -42.50, Subject: "A", Description: ""},
				{Date: d2, Amount: -120.00, Subject: "B", Description: ""},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(dups).To(HaveKey(0))
			Expect(dups).NotTo(HaveKey(1))
		})

		It("differentiates by description", func() {
			d1, _ := model.ParseDate("2024-01-05")
			insertExpense("2024-01-05", -42.50, "A", "desc one")

			dups, err := s.FindDuplicates(ctx, []model.Expense{
				{Date: d1, Amount: -42.50, Subject: "A", Description: "desc two"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(dups).To(BeEmpty())
		})

		It("handles empty slice", func() {
			dups, err := s.FindDuplicates(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(dups).To(BeEmpty())
		})

		It("batches large input correctly", func() {
			d, _ := model.ParseDate("2024-01-05")

			inserted := make([]model.Expense, 200)
			for i := range inserted {
				inserted[i] = model.Expense{
					Date:        d,
					Amount:      float64(-i - 1),
					Subject:     fmt.Sprintf("subj-%d", i),
					Description: fmt.Sprintf("desc-%d", i),
				}
			}
			_, err := s.ImportExpenses(ctx, inserted)
			Expect(err).NotTo(HaveOccurred())

			dups, err := s.FindDuplicates(ctx, inserted)
			Expect(err).NotTo(HaveOccurred())
			Expect(dups).To(HaveLen(200))
		})
	})
})
