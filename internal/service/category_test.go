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

	"github.com/permhoot/pennypinch/internal/model"
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

var _ = Describe("ResolveCategoryParts", func() {
	When("the path is empty", func() {
		It("returns nil", func() {
			Expect(ResolveCategoryParts(nil, "")).To(BeNil())
		})
	})

	DescribeTable("splits a path into one part per component",
		func(path string, want []model.CategoryPart) {
			parts := ResolveCategoryParts(nil, path)
			Expect(parts).To(HaveLen(len(want)))
			for i, w := range want {
				Expect(parts[i].Name).To(Equal(w.Name))
				Expect(parts[i].FullPath).To(Equal(w.FullPath))
				Expect(parts[i].Description).To(BeEmpty())
				Expect(parts[i].Color).To(MatchRegexp(`^#[0-9a-fA-F]{6}$`))
			}
		},
		Entry("single component",
			"income",
			[]model.CategoryPart{{Name: "income", FullPath: "income"}},
		),
		Entry("two components",
			"income/freelance",
			[]model.CategoryPart{
				{Name: "income", FullPath: "income"},
				{Name: "freelance", FullPath: "income/freelance"},
			},
		),
		Entry("three components",
			"income/freelance/consulting",
			[]model.CategoryPart{
				{Name: "income", FullPath: "income"},
				{Name: "freelance", FullPath: "income/freelance"},
				{Name: "consulting", FullPath: "income/freelance/consulting"},
			},
		),
	)

	When("a config entry matches the full path", func() {
		var parts []model.CategoryPart
		var categories []model.Category

		BeforeEach(func() { // Arrange
			categories = []model.Category{
				{Name: "income", Color: "#1a7f37", Description: "Money earned"},
			}
		})

		JustBeforeEach(func() { // Act
			parts = ResolveCategoryParts(categories, "income/freelance")
		})

		It("uses the config color and description for the matching component", func() { // Assert
			Expect(parts[0].Color).To(Equal("#1A7F37"))
			Expect(parts[0].Description).To(Equal("Money earned"))
		})
	})

	When("config entries cover both parent and child paths", func() {
		var parts []model.CategoryPart
		var categories []model.Category

		BeforeEach(func() { // Arrange
			categories = []model.Category{
				{Name: "income", Color: "#000000", Description: "top"},
				{Name: "income/freelance", Color: "#ff00ff", Description: "freelance money"},
			}
		})

		JustBeforeEach(func() { // Act
			parts = ResolveCategoryParts(categories, "income/freelance")
		})

		It("uses the most specific match for each component", func() { // Assert
			Expect(parts[0].Description).To(Equal("top"))
			Expect(parts[1].Description).To(Equal("freelance money"))
			Expect(parts[1].Color).To(Equal("#FF00FF"))
		})
	})

	When("a config entry has a description but no color", func() {
		var parts []model.CategoryPart
		var categories []model.Category

		BeforeEach(func() { // Arrange
			categories = []model.Category{{Name: "income", Description: "x"}}
		})

		JustBeforeEach(func() { // Act
			parts = ResolveCategoryParts(categories, "income")
		})

		It("keeps the description and falls back to the palette color", func() { // Assert
			Expect(parts[0].Color).To(MatchRegexp(`^#[0-9a-fA-F]{6}$`))
			Expect(parts[0].Description).To(Equal("x"))
		})
	})
})
