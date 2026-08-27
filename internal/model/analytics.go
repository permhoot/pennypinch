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

// CategoryTotal is an aggregated total for a single category.
type CategoryTotal struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
	Percent  float64 `json:"percent"`
	Color    string  `json:"color,omitempty"`
}

// MonthlyOverview holds income/expense aggregation for one calendar month.
type MonthlyOverview struct {
	Year              int             `json:"year"`
	Month             int             `json:"month"`
	Income            float64         `json:"income"`
	Expenses          float64         `json:"expenses"`
	Net               float64         `json:"net"`
	ExpenseCategories []CategoryTotal `json:"expense_categories"`
	IncomeCategories  []CategoryTotal `json:"income_categories"`
}

// SankeyNode is a node in a sankey diagram.
type SankeyNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// SankeyLink is a directed link between two sankey nodes.
type SankeyLink struct {
	From  string  `json:"from"`
	To    string  `json:"to"`
	Value float64 `json:"value"`
}

// SankeyData is the full sankey payload.
type SankeyData struct {
	Nodes []SankeyNode `json:"nodes"`
	Links []SankeyLink `json:"links"`
}

// Summary is a set of quick aggregate statistics.
type Summary struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	Net          float64 `json:"net"`
	Count        int     `json:"count"`
	FirstDate    string  `json:"first_date,omitempty"`
	LastDate     string  `json:"last_date,omitempty"`
}
