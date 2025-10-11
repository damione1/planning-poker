package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// MaxCustomValues limits the number of custom vote values
	MaxCustomValues = 20
	// MaxValueLength limits individual value length
	MaxValueLength = 10

	// Template constants - Single source of truth for vote templates
	TemplateModifiedFibonacci = "modified-fibonacci"
	TemplateFibonacci         = "fibonacci"
	TemplateTShirt            = "t-shirt"

	// Template values as comma-separated strings (for forms)
	TemplateModifiedFibonacciValues = "0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100"
	TemplateFibonacciValues         = "1, 2, 3, 5, 8, 13, 21"
	TemplateTShirtValues            = "XXS, XS, S, M, L, XL, XXL"
)

// VoteValidator provides secure validation and parsing for vote values
type VoteValidator struct{}

// NewVoteValidator creates a new vote validator instance
func NewVoteValidator() *VoteValidator {
	return &VoteValidator{}
}

// ParseCustomValues parses comma-separated vote values with validation
// Input: "XS, S, M, L, XL" or "0.5, 1, 2, 3, 5, 8"
// Output: []string{"XS", "S", "M", "L", "XL"} or []string{"0.5", "1", "2", "3", "5", "8"}
func (v *VoteValidator) ParseCustomValues(input string) ([]string, error) {
	if input == "" {
		return nil, fmt.Errorf("custom values cannot be empty")
	}

	// Trim whitespace and split by comma
	input = strings.TrimSpace(input)
	parts := strings.Split(input, ",")

	var values []string
	seen := make(map[string]bool)

	for _, part := range parts {
		// Trim whitespace from each value
		value := strings.TrimSpace(part)

		// Skip empty values
		if value == "" {
			continue
		}

		// Validate value
		if err := v.ValidateValue(value); err != nil {
			return nil, fmt.Errorf("invalid value '%s': %w", value, err)
		}

		// Check for duplicates (case-sensitive)
		if seen[value] {
			return nil, fmt.Errorf("duplicate value: '%s'", value)
		}
		seen[value] = true

		values = append(values, value)
	}

	// Check minimum and maximum count
	if len(values) == 0 {
		return nil, fmt.Errorf("no valid values found")
	}
	if len(values) < 2 {
		return nil, fmt.Errorf("at least 2 values are required (got %d)", len(values))
	}
	if len(values) > MaxCustomValues {
		return nil, fmt.Errorf("too many values (max %d, got %d)", MaxCustomValues, len(values))
	}

	return values, nil
}

// ValidateValue validates a single vote value
func (v *VoteValidator) ValidateValue(value string) error {
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}

	if len(value) > MaxValueLength {
		return fmt.Errorf("value too long (max %d characters)", MaxValueLength)
	}

	// Allow alphanumeric, dots, hyphens, underscores, and spaces
	// This supports: numbers (1, 2, 3), floats (0.5, 1.5), t-shirt sizes (XS, S, M), etc.
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9.\-_ ]+$`)
	if !validPattern.MatchString(value) {
		return fmt.Errorf("contains invalid characters (allowed: letters, numbers, dot, hyphen, underscore, space)")
	}

	// Prevent injection attacks - no control characters
	for _, r := range value {
		if r < 32 || r == 127 {
			return fmt.Errorf("contains control characters")
		}
	}

	return nil
}

// ParseNumericValue attempts to parse a vote value as a number (int or float)
// Returns the float value and true if successful, 0 and false otherwise
func (v *VoteValidator) ParseNumericValue(value string) (float64, bool) {
	// Try parsing as float (handles both int and float)
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		// Validate reasonable range
		if num >= 0 && num <= 1000 {
			return num, true
		}
	}
	return 0, false
}

// IsNumericValue checks if a value can be parsed as a number
func (v *VoteValidator) IsNumericValue(value string) bool {
	_, ok := v.ParseNumericValue(value)
	return ok
}

// GetFibonacciValues returns the default Fibonacci sequence
func (v *VoteValidator) GetFibonacciValues() []string {
	values, _ := v.ParseCustomValues(TemplateFibonacciValues)
	return values
}

// GetModifiedFibonacciValues returns the modified Fibonacci sequence starting from 0.5
func (v *VoteValidator) GetModifiedFibonacciValues() []string {
	values, _ := v.ParseCustomValues(TemplateModifiedFibonacciValues)
	return values
}

// GetTShirtValues returns t-shirt sizing values
func (v *VoteValidator) GetTShirtValues() []string {
	values, _ := v.ParseCustomValues(TemplateTShirtValues)
	return values
}

// GetTemplateValuesString returns the comma-separated values for a template
func (v *VoteValidator) GetTemplateValuesString(templateName string) string {
	switch templateName {
	case TemplateFibonacci:
		return TemplateFibonacciValues
	case TemplateModifiedFibonacci:
		return TemplateModifiedFibonacciValues
	case TemplateTShirt:
		return TemplateTShirtValues
	default:
		return ""
	}
}

// GetPresetTemplate returns preset values for a given template name
func (v *VoteValidator) GetPresetTemplate(templateName string) ([]string, error) {
	switch templateName {
	case TemplateFibonacci:
		return v.GetFibonacciValues(), nil
	case TemplateModifiedFibonacci:
		return v.GetModifiedFibonacciValues(), nil
	case TemplateTShirt:
		return v.GetTShirtValues(), nil
	default:
		return nil, fmt.Errorf("unknown template: %s", templateName)
	}
}

// TemplateInfo holds metadata about a vote template
type TemplateInfo struct {
	ID          string
	Name        string
	Description string
	Values      string
}

// GetAvailableTemplates returns all available preset templates with metadata
func (v *VoteValidator) GetAvailableTemplates() []TemplateInfo {
	return []TemplateInfo{
		{
			ID:          TemplateModifiedFibonacci,
			Name:        "Modified Fibonacci",
			Description: "Modified Fibonacci (" + TemplateModifiedFibonacciValues + ")",
			Values:      TemplateModifiedFibonacciValues,
		},
		{
			ID:          TemplateFibonacci,
			Name:        "Fibonacci",
			Description: "Fibonacci (" + TemplateFibonacciValues + ")",
			Values:      TemplateFibonacciValues,
		},
		{
			ID:          TemplateTShirt,
			Name:        "T-Shirt Sizes",
			Description: "T-Shirt Sizes (" + TemplateTShirtValues + ")",
			Values:      TemplateTShirtValues,
		},
	}
}

// GetAvailableTemplatesMap returns templates as a map (legacy compatibility)
func (v *VoteValidator) GetAvailableTemplatesMap() map[string]string {
	result := make(map[string]string)
	for _, t := range v.GetAvailableTemplates() {
		result[t.ID] = t.Description
	}
	return result
}

// ValidateVoteValue checks if a vote value is valid for a room's pointing method
func (v *VoteValidator) ValidateVoteValue(value string, pointingMethod string, customValues []string) error {
	if value == "" {
		return fmt.Errorf("vote value cannot be empty")
	}

	// Special values always allowed
	if value == "?" || value == "☕" {
		return nil
	}

	switch pointingMethod {
	case "fibonacci":
		fibValues := v.GetFibonacciValues()
		for _, fib := range fibValues {
			if value == fib {
				return nil
			}
		}
		return fmt.Errorf("invalid fibonacci value: '%s'", value)

	case "custom":
		if len(customValues) == 0 {
			return fmt.Errorf("no custom values configured for this room")
		}
		for _, cv := range customValues {
			if value == cv {
				return nil
			}
		}
		return fmt.Errorf("invalid custom value: '%s'", value)

	default:
		return fmt.Errorf("unknown pointing method: '%s'", pointingMethod)
	}
}
