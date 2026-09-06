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

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/permhoot/pennypinch/internal/service"
	"github.com/permhoot/pennypinch/internal/web"
)

var webFS = web.FS

var funcs = template.FuncMap{
	"money": func(v float64) string { return fmt.Sprintf("%.2f", v) },
	"formatAmount": func(v float64, currency string, position string) string {
		amount := fmt.Sprintf("%.2f", v)
		if position == "right" {
			return amount + currency
		}
		return currency + amount
	},
	"top": service.TopLevelCategory,
	"sub": func(a, b int) int { return a - b },
	"add": func(a, b int) int { return a + b },
	"dateval": func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format("2006-01-02")
	},
	"floatval": func(f *float64) string {
		if f == nil {
			return ""
		}
		return strconv.FormatFloat(*f, 'f', 2, 64)
	},
	"pages": pageList,
	"json": func(v any) template.JS {
		b, err := json.Marshal(v)
		if err != nil {
			return template.JS("null")
		}
		return template.JS(b) //nolint:gosec // JSON marshaled data is safe for template.JS
	},
}

// pageList returns a compact window of page numbers for pagination, using 0 to
// represent an ellipsis.
func pageList(current, total int) []int {
	if total <= 7 {
		out := make([]int, total)
		for i := range out {
			out[i] = i + 1
		}
		return out
	}

	start := current - 2
	end := current + 2
	if start < 1 {
		start = 1
		end = start + 4
	}
	if end > total {
		end = total
		start = end - 4
	}
	if start < 1 {
		start = 1
	}

	out := []int{}
	if start > 1 {
		out = append(out, 1)
		if start > 2 {
			out = append(out, 0)
		}
	}
	for p := start; p <= end; p++ {
		out = append(out, p)
	}
	if end < total {
		if end < total-1 {
			out = append(out, 0)
		}
		out = append(out, total)
	}
	return out
}

type pageTemplate struct {
	tmpl *template.Template
}

func parsePage(files ...string) (pageTemplate, error) {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = "templates/" + f + ".html"
	}
	t, err := template.New("").Funcs(funcs).ParseFS(web.FS, paths...)
	if err != nil {
		return pageTemplate{}, err
	}
	return pageTemplate{tmpl: t}, nil
}

func (p pageTemplate) render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := p.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}
