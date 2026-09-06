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

package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/permhoot/pennypinch/internal/model"
)

var _ = Describe("Manager", func() {
	var path string

	BeforeEach(func() {
		path = filepath.Join(GinkgoT().TempDir(), "config.yaml")
	})

	Describe("LoadManager", func() {
		Context("when no config file exists", func() {
			var m *Manager
			var err error

			JustBeforeEach(func() {
				m, err = LoadManager(path)
				if m != nil {
					DeferCleanup(m.Close)
				}
			})

			It("does not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("uses the default theme", func() {
				Expect(m.Snapshot().Settings.Theme).To(Equal("dark"))
			})

			It("has no categories", func() {
				Expect(m.Snapshot().Categories).To(BeEmpty())
			})

			It("creates the config file on disk", func() {
				Expect(path).To(BeAnExistingFile())
			})
		})

		When("the config file is malformed", func() {
			var err error

			BeforeEach(func() {
				Expect(os.WriteFile(path, []byte("categories: [not-a-map"), 0o644)).To(Succeed())
			})

			JustBeforeEach(func() {
				_, err = LoadManager(path)
			})

			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("EnsureCategories", func() {
		var m *Manager

		BeforeEach(func() {
			var err error
			m, err = LoadManager(path)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(m.Close)
		})

		Context("adding new categories", func() {
			var added []model.Category
			var err error

			JustBeforeEach(func() {
				added, err = m.EnsureCategories([]string{"food/groceries", "transportation/fuel"})
			})

			It("does not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the added categories", func() {
				Expect(added).To(HaveLen(2))
			})

			It("preserves the category name", func() {
				Expect(added[0].Name).To(Equal("food/groceries"))
			})
		})

		Context("with duplicates of existing categories", func() {
			var added []model.Category
			var err error

			BeforeEach(func() {
				_, err := m.EnsureCategories([]string{"food/groceries", "transportation/fuel"})
				Expect(err).NotTo(HaveOccurred())
			})

			JustBeforeEach(func() {
				added, err = m.EnsureCategories([]string{"food/groceries", "food/groceries", "new/cat"})
			})

			It("does not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns only the new category", func() {
				Expect(added).To(HaveLen(1))
			})

			It("has the correct name for the new category", func() {
				Expect(added[0].Name).To(Equal("new/cat"))
			})
		})

		Context("persistence after adding categories", func() {
			var reloaded *Config

			BeforeEach(func() {
				_, err := m.EnsureCategories([]string{"food/groceries", "transportation/fuel", "new/cat"})
				Expect(err).NotTo(HaveOccurred())

				fresh, err := LoadManager(path)
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(fresh.Close)

				reloaded = fresh.Snapshot()
			})

			It("persists all categories to disk", func() {
				Expect(reloaded.Categories).To(HaveLen(3))
			})

			It("assigns a color to every category", func() {
				for _, c := range reloaded.Categories {
					Expect(c.Color).NotTo(BeEmpty())
				}
			})
		})
	})

	Describe("RestoreCategories", func() {
		var m *Manager
		var before []model.Category

		BeforeEach(func() {
			var err error
			m, err = LoadManager(path)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(m.Close)

			before = m.Snapshot().Categories
		})

		Context("after adding categories", func() {
			var err error

			BeforeEach(func() {
				_, err := m.EnsureCategories([]string{"foo"})
				Expect(err).NotTo(HaveOccurred())
				Expect(m.Snapshot().Categories).To(HaveLen(1))
			})

			JustBeforeEach(func() {
				err = m.RestoreCategories(before)
			})

			It("does not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("restores the original empty category list", func() {
				Expect(m.Snapshot().Categories).To(BeEmpty())
			})
		})
	})
})
