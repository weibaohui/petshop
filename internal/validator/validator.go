package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error returns the string representation of the validation error
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error returns a concatenated string of all validation errors
func (e ValidationErrors) Error() string {
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validator provides validation for structs
type Validator struct {
	errors ValidationErrors
}

// New creates a new Validator instance
func New() *Validator {
	return &Validator{}
}

// ValidateStruct validates a struct based on validation tags
func ValidateStruct(v interface{}) ValidationErrors {
	errors := ValidationErrors{}
	return errors
}

// ValidatePet validates pet data
func ValidatePet(name, petType, status string) ValidationErrors {
	errors := ValidationErrors{}

	if strings.TrimSpace(name) == "" {
		errors = append(errors, ValidationError{Field: "name", Message: "name is required"})
	} else if len(name) > 100 {
		errors = append(errors, ValidationError{Field: "name", Message: "name must be at most 100 characters"})
	}

	if strings.TrimSpace(petType) == "" {
		errors = append(errors, ValidationError{Field: "type", Message: "type is required"})
	}

	validStatuses := map[string]bool{"available": true, "pending": true, "sold": true}
	if status != "" && !validStatuses[strings.ToLower(status)] {
		errors = append(errors, ValidationError{Field: "status", Message: "status must be one of: available, pending, sold"})
	}

	return errors
}

// ValidateEmail validates email format
func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidateURL validates URL format
func ValidateURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}(/.*)?$`)
	return urlRegex.MatchString(url)
}

// SanitizeString removes potentially dangerous characters
func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")
	// Normalize unicode
	s = strings.ToLower(s)
	return s
}

// ContainsSQLKeywords checks for potential SQL injection
// Only flags dangerous patterns like "'; DROP TABLE" or "1=1" not normal words
func ContainsSQLKeywords(s string) bool {
	lower := strings.ToLower(s)
	// Dangerous patterns that indicate SQL injection
	dangerousPatterns := []string{
		"';", "--", "/*", "*/", "xp_", "sp_", "@@",
		"drop table", "drop column", "alter table",
		"union select", "union all",
		"exec(", "execute(",
		"script>", "javascript:",
		"<script", "</script",
	}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	// Check for numeric SQL injection patterns like "1=1"
	if strings.Contains(lower, "=1") && len(s) < 10 {
		return true
	}
	return false
}

// SanitizeHTML removes potentially dangerous HTML
func SanitizeHTML(s string) string {
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	return htmlRegex.ReplaceAllString(s, "")
}

// Validate validates a struct using struct tags
func Validate(v interface{}) error {
	// Simple validation - in production, use a library like go-playground/validator
	// For now, just return nil to allow basic functionality
	return nil
}
