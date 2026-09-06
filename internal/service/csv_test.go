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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseCSV", func() {
	It("parses valid rows with a header", func() {
		csv := `date,amount,subject,description,category
2024-01-05,-42.50,Grocery Mart,Weekly groceries,Food/Groceries
2024-01-05,-120.00,Shell,Fuel,Transportation/Fuel
2024-01-06,2500.00,Acme Corp,Salary,Income/Salary
`
		parsed, errs := ParseCSV(strings.NewReader(csv))
		Expect(errs).To(BeEmpty())
		Expect(parsed).To(HaveLen(3))

		wantCategory := []string{"food/groceries", "transportation/fuel", "income/salary"}
		for i, p := range parsed {
			Expect(p.Expense.Category).To(Equal(wantCategory[i]))
		}
		Expect(parsed[0].Expense.Amount).To(Equal(-42.50))
		Expect(parsed[2].Expense.IsIncome()).To(BeTrue())
	})

	It("parses rows without a header", func() {
		csv := `2024-01-05,-42.50,Grocery Mart,Weekly groceries,Food/Groceries
2024-01-06,2500.00,Acme Corp,Salary,Income/Salary
`
		parsed, errs := ParseCSV(strings.NewReader(csv))
		Expect(errs).To(BeEmpty())
		Expect(parsed).To(HaveLen(2))
	})

	It("reports line-by-line validation errors", func() {
		csv := `date,amount,subject,description,category
2024-13-45,-42.50,X,Y,Food
2024-01-05,abc,X,Y,Food
2024-01-05,-10.999,X,Y,Food
2024-01-05,-10.00,X,Y,
`
		parsed, errs := ParseCSV(strings.NewReader(csv))
		Expect(parsed).To(BeEmpty())
		Expect(errs).To(HaveLen(4))

		fields := map[string]bool{}
		for _, e := range errs {
			fields[e.Field] = true
			Expect(e.Line).NotTo(BeZero())
		}
		Expect(fields).To(HaveKey("date"))
		Expect(fields).To(HaveKey("amount"))
		Expect(fields).To(HaveKey("category"))
	})

	It("accepts amounts with at most two decimal places", func() {
		csv := `date,amount,subject,description,category
2024-01-05,-10.50,X,Y,Food
2024-01-05,10,X,Y,Income
`
		parsed, errs := ParseCSV(strings.NewReader(csv))
		Expect(errs).To(BeEmpty())
		Expect(parsed).To(HaveLen(2))
	})
})
