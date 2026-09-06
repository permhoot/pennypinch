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
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/permhoot/pennypinch/internal/model"
)

// CSVColumnNames is the fixed import column order.
var CSVColumnNames = []string{"date", "amount", "subject", "description", "category"}

// ImportError describes a single row-level validation failure.
type ImportError struct {
	Line    int    `json:"line"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ParsedExpense is a validated expense ready for import.
type ParsedExpense struct {
	Expense model.Expense
	Line    int
}

// CSVTemplate returns example rows and column explanations for the download.
func CSVTemplate() string {
	var b strings.Builder
	b.WriteString("date,amount,subject,description,category\n")
	b.WriteString("2024-01-05,-42.50,Grocery Mart,Weekly groceries,food/groceries\n")
	b.WriteString("2024-01-05,-120.00,Shell,Fuel for car,transportation/fuel\n")
	b.WriteString("2024-01-06,2500.00,Acme Corp,January salary,income/salary\n")
	return b.String()
}

// ParseCSV reads and validates a CSV stream in the fixed column order
// date,amount,subject,description,category. It returns parsed rows and any
// validation errors; the caller must reject the whole import if errors exist.
func ParseCSV(r io.Reader) ([]ParsedExpense, []ImportError) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = 5
	reader.TrimLeadingSpace = true

	var parsed []ParsedExpense
	var errs []ImportError

	line := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			errs = append(errs, ImportError{Line: line, Message: fmt.Sprintf("malformed CSV: %v", err)})
			continue
		}

		// Skip the header row when it matches the expected column names.
		if line == 1 && isHeader(record) {
			continue
		}

		pe, rowErrs := validateRecord(line, record)
		if len(rowErrs) > 0 {
			errs = append(errs, rowErrs...)
			continue
		}
		parsed = append(parsed, pe)
	}

	return parsed, errs
}

func isHeader(record []string) bool {
	if len(record) != len(CSVColumnNames) {
		return false
	}
	for i, h := range CSVColumnNames {
		if !strings.EqualFold(strings.TrimSpace(record[i]), h) {
			return false
		}
	}
	return true
}

func validateRecord(line int, record []string) (ParsedExpense, []ImportError) {
	var errs []ImportError
	add := func(field, msg string) {
		errs = append(errs, ImportError{Line: line, Field: field, Message: msg})
	}

	dateStr := strings.TrimSpace(record[0])
	date, err := model.ParseDate(dateStr)
	if err != nil {
		add("date", err.Error())
	}

	amountStr := strings.TrimSpace(record[1])
	amount, err := parseAmount(amountStr)
	if err != nil {
		add("amount", err.Error())
	}

	subject := strings.TrimSpace(record[2])
	description := strings.TrimSpace(record[3])

	category, err := NormalizeCategoryPath(record[4])
	if err != nil {
		add("category", err.Error())
	}

	if len(errs) > 0 {
		return ParsedExpense{}, errs
	}

	return ParsedExpense{
		Expense: model.Expense{
			Date:        date,
			Amount:      amount,
			Subject:     subject,
			Description: description,
			Category:    category,
		},
		Line: line,
	}, nil
}

func parseAmount(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("amount is required")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: must be a number", s)
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid amount %q", s)
	}
	if !hasMaxTwoDecimals(s) {
		return 0, fmt.Errorf("amount %q must have at most 2 decimal places", s)
	}
	return v, nil
}

func hasMaxTwoDecimals(s string) bool {
	dot := strings.IndexByte(s, '.')
	if dot < 0 {
		return true
	}
	frac := s[dot+1:]
	// Strip a possible exponent suffix like "1.23e2".
	if e := strings.IndexAny(frac, "eE"); e >= 0 {
		frac = frac[:e]
	}
	return len(frac) <= 2
}
