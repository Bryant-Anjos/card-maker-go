package model

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// SchemaVersion is the data contract version understood by this package.
const SchemaVersion = 1

var (
	identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
	localePattern     = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
	hexColorPattern   = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// problems collects human-readable validation failures, each prefixed by the
// field path it refers to.
type problems []string

func (p *problems) addf(field, format string, args ...any) {
	*p = append(*p, field+": "+fmt.Sprintf(format, args...))
}

// addNested records the problems of a child value under a path prefix.
func (p *problems) addNested(prefix string, child problems) {
	for _, c := range child {
		*p = append(*p, prefix+"."+c)
	}
}

func (p problems) err() error {
	if len(p) == 0 {
		return nil
	}

	errs := make([]error, len(p))
	for i, msg := range p {
		errs[i] = errors.New(msg)
	}

	return errors.Join(errs...)
}

func checkSchemaVersion(p *problems, version int) {
	if version != SchemaVersion {
		p.addf("schemaVersion", "unsupported version %d, expected %d", version, SchemaVersion)
	}
}

func checkIdentifier(p *problems, field, value string) {
	if !identifierPattern.MatchString(value) {
		p.addf(field, "%q must match %s", value, identifierPattern)
	}
}

// IsValidLocale reports whether code looks like a locale code such as "pt-BR".
func IsValidLocale(code string) bool {
	return localePattern.MatchString(code)
}

func checkLocale(p *problems, field, value string) {
	if !IsValidLocale(value) {
		p.addf(field, "invalid locale code %q", value)
	}
}

// checkRelativePath rejects empty, absolute and root-escaping paths so that
// user-provided data can never point outside the theme or output directory.
// Paths use forward slashes and may not contain empty, "." or ".." segments;
// the JSON Schema "relativePath" definition enforces the same rule.
func checkRelativePath(p *problems, field, value string) {
	if value == "" {
		p.addf(field, "is required")
		return
	}
	if !isSafeRelativePath(value) {
		p.addf(field, "%q must be a relative path inside its root directory", value)
	}
}

func isSafeRelativePath(value string) bool {
	if strings.Contains(value, `\`) || !filepath.IsLocal(value) {
		return false
	}

	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}

	return true
}

func checkPositive(p *problems, field string, value int) {
	if value <= 0 {
		p.addf(field, "must be positive, got %d", value)
	}
}
