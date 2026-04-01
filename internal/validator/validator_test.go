package validator

import (
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{Field: "email", Message: "invalid format"}
	expected := "email: invalid format"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errors := ValidationErrors{
		{Field: "email", Message: "invalid format"},
		{Field: "name", Message: "required"},
	}
	expected := "email: invalid format; name: required"
	if errors.Error() != expected {
		t.Errorf("expected %q, got %q", expected, errors.Error())
	}
}

func TestValidationErrors_HasErrors(t *testing.T) {
	empty := ValidationErrors{}
	if empty.HasErrors() {
		t.Error("expected empty errors to have no errors")
	}

	withErrors := ValidationErrors{{Field: "email", Message: "invalid"}}
	if !withErrors.HasErrors() {
		t.Error("expected non-empty errors to have errors")
	}
}

func TestNew(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestValidatePet(t *testing.T) {
	tests := []struct {
		name     string
		petName  string
		petType  string
		status   string
		errCount int
	}{
		{"Valid pet", "Fluffy", "dog", "available", 0},
		{"Empty name", "", "dog", "available", 1},
		{"Empty type", "Fluffy", "", "available", 1},
		{"Invalid status", "Fluffy", "dog", "invalid", 1},
		{"Both empty", "", "", "", 2},
		{"Long name", string(make([]byte, 101)), "dog", "available", 1},
		{"Valid all statuses", "Fluffy", "dog", "AVAILABLE", 0},
		{"Valid pending", "Fluffy", "cat", "PENDING", 0},
		{"Valid sold", "Fluffy", "bird", "SOLD", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidatePet(tt.petName, tt.petType, tt.status)
			if len(errors) != tt.errCount {
				t.Errorf("expected %d errors, got %d: %v", tt.errCount, len(errors), errors)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name@domain.org", true},
		{"user+tag@example.co.uk", true},
		{"invalid", false},
		{"@domain.com", false},
		{"user@", false},
		{"user@domain", false},
		{"user name@domain.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := ValidateEmail(tt.email)
			if result != tt.expected {
				t.Errorf("ValidateEmail(%q) = %v, expected %v", tt.email, result, tt.expected)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"http://example.com/path", true},
		{"https://example.com/path/to/page", true},
		{"http://example.com/path?query=value", true},
		{"invalid", false},
		{"example.com", false},
		{"ftp://example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := ValidateURL(tt.url)
			if result != tt.expected {
				t.Errorf("ValidateURL(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"Hello", "hello"},
		{"hello\x00world", "helloworld"},
		{"HELLO", "hello"},
		{"  ", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainsSQLKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"normal text", false},
		{"'; DROP TABLE users", true},
		{"admin'--", true},
		{"1=1", true},
		{"test", false},
		{"xp_cmdshell", true},
		{"sp_executesql", true},
		{"union select", true},
		{"exec(user)", true},
		{"/* comment */", true},
		{"*/", true},
		{"javascript:alert(1)", true},
		{"<script>alert(1)</script>", true},
		{"</script>", true},
		{"@@version", true},
		{"drop table", true},
		{"alter table", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ContainsSQLKeywords(tt.input)
			if result != tt.expected {
				t.Errorf("ContainsSQLKeywords(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>hello</p>", "hello"},
		{"<script>alert(1)</script>", "alert(1)"},
		{"<div class='test'><span>content</span></div>", "content"},
		{"plain text", "plain text"},
		{"", ""},
		{"<>", ""},
		{"a<b>c", "ac"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeHTML(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
