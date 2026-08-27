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
)

var _ = Describe("Manager", func() {
	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
	})

	Describe("LoadManager", func() {
		It("creates a default config file when missing", func() {
			path := filepath.Join(dir, "config.yaml")
			m, err := LoadManager(path)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(m.Close)

			cfg := m.Snapshot()
			Expect(cfg.Settings.Theme).To(Equal("dark"))
			Expect(cfg.Categories).To(BeEmpty())
			Expect(path).To(BeAnExistingFile())
		})

		It("returns an error for a malformed config", func() {
			path := filepath.Join(dir, "config.yaml")
			Expect(os.WriteFile(path, []byte("categories: [not-a-map"), 0o644)).To(Succeed())

			_, err := LoadManager(path)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("EnsureCategories", func() {
		It("assigns colors, deduplicates, and persists", func() {
			path := filepath.Join(dir, "config.yaml")
			m, err := LoadManager(path)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(m.Close)

			added, err := m.EnsureCategories([]string{"food/groceries", "transportation/fuel"})
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(HaveLen(2))
			Expect(added[0].Name).To(Equal("food/groceries"))

			added2, err := m.EnsureCategories([]string{"food/groceries", "food/groceries", "new/cat"})
			Expect(err).NotTo(HaveOccurred())
			Expect(added2).To(HaveLen(1))
			Expect(added2[0].Name).To(Equal("new/cat"))

			m2, err := LoadManager(path)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(m2.Close)

			cfg := m2.Snapshot()
			Expect(cfg.Categories).To(HaveLen(3))
			for _, c := range cfg.Categories {
				Expect(c.Color).NotTo(BeEmpty())
			}
		})
	})

	Describe("RestoreCategories", func() {
		It("rolls back to the prior category list", func() {
			path := filepath.Join(dir, "config.yaml")
			m, err := LoadManager(path)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(m.Close)

			before := m.Snapshot().Categories
			_, err = m.EnsureCategories([]string{"foo"})
			Expect(err).NotTo(HaveOccurred())
			Expect(m.Snapshot().Categories).To(HaveLen(1))

			Expect(m.RestoreCategories(before)).To(Succeed())
			Expect(m.Snapshot().Categories).To(BeEmpty())
		})
	})
})
