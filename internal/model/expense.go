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
package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// Date is a calendar date (no time component) serialized as YYYY-MM-DD.
type Date struct {
	time.Time
}

// NewDate builds a Date from a year/month/day triple, normalized to UTC midnight.
func NewDate(year int, month time.Month, day int) Date {
	return Date{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// ParseDate parses a YYYY-MM-DD string.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q: must be YYYY-MM-DD", s)
	}
	return Date{Time: t}, nil
}

func (d Date) String() string {
	return d.Time.Format("2006-01-02")
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Expense is a single transaction. Amount is signed: negative for expenses,
// positive for income. Category is a slash-separated canonical path.
type Expense struct {
	ID          int64   `json:"id"`
	Date        Date    `json:"date"`
	Amount      float64 `json:"amount"`
	Subject     string  `json:"subject"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
}

// IsIncome reports whether the amount represents income (positive).
func (e Expense) IsIncome() bool {
	return e.Amount > 0
}

// IsExpense reports whether the amount represents an expense (negative).
func (e Expense) IsExpense() bool {
	return e.Amount < 0
}
