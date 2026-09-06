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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NormalizeCategoryPath", func() {
	DescribeTable("normalizes valid paths",
		func(input, want string) {
			got, err := NormalizeCategoryPath(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(want))
		},
		Entry("lowercases", "Food/Groceries", "food/groceries"),
		Entry("trims outer whitespace", "  Food / Groceries  ", "food/groceries"),
		Entry("unlimited nesting", "Food/Groceries/Supermarket", "food/groceries/supermarket"),
		Entry("top-level only", "Food", "food"),
		Entry("mixed case", "FOOD/Groceries", "food/groceries"),
		Entry("spaces around slash", "Food   /   Groceries", "food/groceries"),
		Entry("collapses internal whitespace", "Food  Groceries", "food groceries"),
	)

	DescribeTable("rejects invalid paths",
		func(input string) {
			_, err := NormalizeCategoryPath(input)
			Expect(err).To(HaveOccurred())
		},
		Entry("empty", ""),
		Entry("whitespace only", "   "),
		Entry("trailing slash", "Food/"),
		Entry("leading slash", "/Food"),
		Entry("double slash", "Food//Groceries"),
		Entry("empty component", "Food/ /Groceries"),
	)
})

var _ = Describe("TopLevelCategory", func() {
	DescribeTable("extracts the first component",
		func(input, want string) {
			Expect(TopLevelCategory(input)).To(Equal(want))
		},
		Entry("three levels", "food/groceries/supermarket", "food"),
		Entry("two levels", "food/groceries", "food"),
		Entry("single level", "food", "food"),
		Entry("transportation", "transportation/fuel", "transportation"),
	)
})
