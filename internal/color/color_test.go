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

package color

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NormalizeColor", func() {
	DescribeTable("valid colors",
		func(input, want string) {
			Expect(NormalizeColor(input)).To(Equal(want))
		},
		Entry("lowercase hex", "#ff0000", "#FF0000"),
		Entry("uppercase hex", "#FF0000", "#FF0000"),
		Entry("short hex expands", "#f00", "#FF0000"),
		Entry("named color", "red", "#FF0000"),
		Entry("case-insensitive name", "RED", "#FF0000"),
		Entry("trims whitespace", "  blue  ", "#0000FF"),
		Entry("full hex preserved", "#0000FF", "#0000FF"),
		Entry("short hex abc", "#abc", "#AABBCC"),
	)

	DescribeTable("invalid colors",
		func(input string) {
			_, err := NormalizeColor(input)
			Expect(err).To(HaveOccurred())
		},
		Entry("wrong length", "#12"),
		Entry("wrong length 5", "#12345"),
		Entry("non-hex digits", "#GGGGGG"),
		Entry("unknown name", "notacolor"),
		Entry("empty", ""),
	)
})

var _ = Describe("PaletteColor", func() {
	It("is stable for the same key", func() {
		Expect(PaletteColor("food", DefaultPalette)).To(Equal(PaletteColor("food", DefaultPalette)))
	})
})

var _ = Describe("NextPaletteColor", func() {
	It("returns distinct sequential colors", func() {
		Expect(NextPaletteColor(0)).NotTo(Equal(NextPaletteColor(1)))
	})

	It("starts at the first palette color", func() {
		Expect(NextPaletteColor(0)).To(Equal(DefaultPalette[0]))
	})

	It("cycles after the palette is exhausted", func() {
		Expect(NextPaletteColor(len(DefaultPalette))).To(Equal(DefaultPalette[0]))
	})
})
