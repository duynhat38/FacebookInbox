package utils

import (
	"errors"
	"strings"

	"facebook-inbox/internal/domain"
)

// Validation error messages use stable English keys; the frontend maps them to
// localized strings. Keeping them as sentinel errors lets callers compare and
// test them without matching on free text.
var (
	ErrThreadsInvalid        = errors.New("validation.threads")
	ErrDelayInvalid          = errors.New("validation.delay")
	ErrMaxErrorsInvalid      = errors.New("validation.maxErrors")
	ErrPagesPerThreadInvalid = errors.New("validation.pagesPerThread")
	ErrAccountPerLinkInvalid = errors.New("validation.accountPerLink")
)

// ValidateSettings enforces the numeric constraints from the specification.
// It returns the first violation encountered, or nil when the settings are
// valid.
func ValidateSettings(s domain.Settings) error {
	if s.Threads < 1 {
		return ErrThreadsInvalid
	}
	if s.DelaySeconds < 0 {
		return ErrDelayInvalid
	}
	if s.MaxErrors < 1 {
		return ErrMaxErrorsInvalid
	}
	if s.PagesPerThread < 1 {
		return ErrPagesPerThreadInvalid
	}
	if s.AccountPerLink < 1 {
		return ErrAccountPerLinkInvalid
	}
	return nil
}

// IsImageExt reports whether the lower-cased extension (including the dot) is a
// supported image type.
func IsImageExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".webp", ".pdf":
		return true
	default:
		return false
	}
}
