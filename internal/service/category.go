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
	"fmt"
	"strings"
)

// NormalizeCategoryPath trims outer whitespace, splits on "/", trims and
// collapses internal whitespace in each component, rejects empty components,
// lowercases the result, and rejoins with "/". The returned value is the
// canonical category key used for storage, matching, and config lookup.
func NormalizeCategoryPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("category cannot be empty")
	}

	parts := strings.Split(trimmed, "/")
	components := make([]string, 0, len(parts))
	for _, p := range parts {
		comp := strings.Join(strings.Fields(p), " ")
		if comp == "" {
			return "", fmt.Errorf("category %q contains an empty component", path)
		}
		components = append(components, strings.ToLower(comp))
	}
	return strings.Join(components, "/"), nil
}

// TopLevelCategory returns the first component before the first slash, or the
// whole string when there is no slash. Input must already be normalized; the
// result is returned unchanged otherwise.
func TopLevelCategory(normalizedPath string) string {
	if i := strings.IndexByte(normalizedPath, '/'); i >= 0 {
		return normalizedPath[:i]
	}
	return normalizedPath
}

// Components splits a normalized category path into its components.
func Components(normalizedPath string) []string {
	if normalizedPath == "" {
		return nil
	}
	return strings.Split(normalizedPath, "/")
}

// Parent returns the normalized path one level up, or "" for top-level paths.
func Parent(normalizedPath string) string {
	comps := Components(normalizedPath)
	if len(comps) <= 1 {
		return ""
	}
	return strings.Join(comps[:len(comps)-1], "/")
}
