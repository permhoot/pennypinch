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

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/permhoot/pennypinch/internal/model"
)

const schema = `
CREATE TABLE IF NOT EXISTS expenses (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	date        TEXT NOT NULL,
	amount      REAL NOT NULL,
	subject     TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	category    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date);
CREATE INDEX IF NOT EXISTS idx_expenses_category ON expenses(category);
CREATE INDEX IF NOT EXISTS idx_expenses_date_category ON expenses(date, category);
`

// SQLite implements Storage backed by a SQLite database in WAL mode.
type SQLite struct {
	db *sql.DB
}

// OpenSQLite opens (creating if needed) a SQLite database at path and enables
// WAL mode for better concurrent reads.
func OpenSQLite(path string) (*SQLite, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=OFF;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set pragma: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &SQLite{db: db}, nil
}

func (s *SQLite) Close() error { return s.db.Close() }

func scanExpense(row interface{ Scan(...any) error }) (*model.Expense, error) {
	var e model.Expense
	var dateStr string
	if err := row.Scan(&e.ID, &dateStr, &e.Amount, &e.Subject, &e.Description, &e.Category); err != nil {
		return nil, err
	}
	d, err := model.ParseDate(dateStr)
	if err != nil {
		return nil, err
	}
	e.Date = d
	return &e, nil
}

func expenseArgs(e *model.Expense) []any {
	return []any{e.Date.String(), e.Amount, e.Subject, e.Description, e.Category}
}

func (s *SQLite) CreateExpense(ctx context.Context, e *model.Expense) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO expenses (date, amount, subject, description, category) VALUES (?, ?, ?, ?, ?)`,
		expenseArgs(e)...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLite) GetExpense(ctx context.Context, id int64) (*model.Expense, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, date, amount, subject, description, category FROM expenses WHERE id = ?`, id)
	e, err := scanExpense(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	return e, err
}

func (s *SQLite) UpdateExpense(ctx context.Context, e *model.Expense) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE expenses SET date = ?, amount = ?, subject = ?, description = ?, category = ? WHERE id = ?`,
		e.Date.String(), e.Amount, e.Subject, e.Description, e.Category, e.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLite) DeleteExpense(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM expenses WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLite) DeleteExpenses(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM expenses WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

var sortColumns = map[string]string{
	"date":        "date",
	"amount":      "amount",
	"subject":     "subject",
	"description": "description",
	"category":    "category",
}

func buildListQuery(f Filter) (string, []any, string, []any) {
	var where []string
	var args []any

	if f.Search != "" {
		where = append(where, `(subject LIKE ? OR description LIKE ? OR category LIKE ?)`)
		like := "%" + f.Search + "%"
		args = append(args, like, like, like)
	}
	if f.Category != "" {
		where = append(where, `(category = ? OR category LIKE ?)`)
		args = append(args, f.Category, f.Category+"/%")
	}
	if f.DateFrom != nil {
		where = append(where, `date >= ?`)
		args = append(args, f.DateFrom.Format("2006-01-02"))
	}
	if f.DateTo != nil {
		where = append(where, `date <= ?`)
		args = append(args, f.DateTo.Format("2006-01-02"))
	}
	if f.AmountMin != nil {
		where = append(where, `amount >= ?`)
		args = append(args, *f.AmountMin)
	}
	if f.AmountMax != nil {
		where = append(where, `amount <= ?`)
		args = append(args, *f.AmountMax)
	}

	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	order := "date DESC, id DESC"
	if col, ok := sortColumns[f.SortBy]; ok {
		dir := "ASC"
		if strings.EqualFold(f.SortDir, "desc") {
			dir = "DESC"
		}
		order = col + " " + dir + ", id " + dir
	}

	countSQL := "SELECT COUNT(*) FROM expenses" + clause
	listSQL := "SELECT id, date, amount, subject, description, category FROM expenses" +
		clause + " ORDER BY " + order

	page := f.Page
	if page < 1 {
		page = 1
	}
	size := f.PageSize
	if size < 1 {
		size = 50
	}
	limit := size
	offset := (page - 1) * size

	listSQL += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	return countSQL, args, listSQL, args
}

func (s *SQLite) ListExpenses(ctx context.Context, f Filter) ([]model.Expense, int64, error) {
	countSQL, args, listSQL, listArgs := buildListQuery(f)

	var total int64
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = rows.Close()
	}()

	expenses := make([]model.Expense, 0)
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, 0, err
		}
		expenses = append(expenses, *e)
	}
	return expenses, total, rows.Err()
}

func (s *SQLite) AllExpenses(ctx context.Context) ([]model.Expense, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, date, amount, subject, description, category FROM expenses ORDER BY date ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	expenses := make([]model.Expense, 0)
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, *e)
	}
	return expenses, rows.Err()
}

func (s *SQLite) ImportExpenses(ctx context.Context, expenses []model.Expense) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO expenses (date, amount, subject, description, category) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = stmt.Close()
	}()

	for i := range expenses {
		if _, err := stmt.ExecContext(ctx, expenseArgs(&expenses[i])...); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int64(len(expenses)), nil
}

func (s *SQLite) FindDuplicates(ctx context.Context, expenses []model.Expense) (map[int]bool, error) {
	dups := make(map[int]bool)

	if len(expenses) == 0 {
		return dups, nil
	}

	const batchSize = 150
	var sb strings.Builder

	for start := 0; start < len(expenses); start += batchSize {
		end := start + batchSize
		if end > len(expenses) {
			end = len(expenses)
		}

		batch := expenses[start:end]

		sb.Reset()
		sb.WriteString(`WITH candidates(idx, date, amount, subject, description) AS (VALUES `)

		args := make([]any, 0, len(batch)*5)
		for i := range batch {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("(?, ?, ?, ?, ?)")
			args = append(args, start+i, batch[i].Date.String(), batch[i].Amount, batch[i].Subject, batch[i].Description)
		}

		sb.WriteString(`) SELECT c.idx FROM candidates c WHERE EXISTS (SELECT 1 FROM expenses e WHERE e.date = c.date AND e.amount = c.amount AND e.subject = c.subject AND e.description = c.description)`)

		if err := s.findDuplicateBatch(ctx, sb.String(), args, dups); err != nil {
			return nil, err
		}
	}

	return dups, nil
}

func (s *SQLite) findDuplicateBatch(ctx context.Context, query string, args []any, dups map[int]bool) error {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var idx int
		if err := rows.Scan(&idx); err != nil {
			return err
		}
		dups[idx] = true
	}

	return rows.Err()
}

func (s *SQLite) DistinctCategories(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT category FROM expenses ORDER BY category ASC`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var cats []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		if c != "" {
			cats = append(cats, c)
		}
	}
	return cats, rows.Err()
}

func (s *SQLite) MonthlyTotals(ctx context.Context, year, month int) (map[string]model.CategoryTotal, int64, error) {
	return s.categoryTotals(ctx, "strftime('%m', date) = ? AND strftime('%Y', date) = ?",
		fmt.Sprintf("%02d", month), fmt.Sprintf("%04d", year))
}

func (s *SQLite) YearlyTotals(ctx context.Context, year int) (map[string]model.CategoryTotal, int64, error) {
	return s.categoryTotals(ctx, "strftime('%Y', date) = ?", fmt.Sprintf("%04d", year))
}

func (s *SQLite) YearlyIncomeBySubject(ctx context.Context, year int) (map[string]float64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT subject, SUM(amount) FROM expenses
		 WHERE strftime('%Y', date) = ? AND amount > 0
		 GROUP BY subject`, fmt.Sprintf("%04d", year))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make(map[string]float64)
	for rows.Next() {
		var subject string
		var sum float64
		if err := rows.Scan(&subject, &sum); err != nil {
			return nil, err
		}
		if subject == "" {
			subject = "other"
		}
		out[subject] += sum
	}
	return out, rows.Err()
}

func (s *SQLite) categoryTotals(ctx context.Context, clause string, args ...any) (map[string]model.CategoryTotal, int64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT category, SUM(amount), COUNT(*) FROM expenses WHERE `+clause+` GROUP BY category`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = rows.Close()
	}()

	totals := make(map[string]model.CategoryTotal)
	var count int64
	for rows.Next() {
		var cat string
		var sum float64
		var n int64
		if err := rows.Scan(&cat, &sum, &n); err != nil {
			return nil, 0, err
		}
		if cat == "" {
			cat = "uncategorized"
		}
		totals[cat] = model.CategoryTotal{Category: cat, Total: sum, Count: int(n)}
		count += n
	}
	return totals, count, rows.Err()
}

func (s *SQLite) Summary(ctx context.Context) (model.Summary, error) {
	var sum model.Summary
	var firstDate sql.NullString
	var lastDate sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0),
			COALESCE(SUM(amount), 0),
			COUNT(*),
			MIN(date),
			MAX(date)
		FROM expenses`).Scan(&sum.TotalIncome, &sum.TotalExpense, &sum.Net, &sum.Count, &firstDate, &lastDate)
	if err != nil {
		return sum, err
	}
	if firstDate.Valid {
		sum.FirstDate = firstDate.String
	}
	if lastDate.Valid {
		sum.LastDate = lastDate.String
	}
	return sum, nil
}

var _ Storage = (*SQLite)(nil)
