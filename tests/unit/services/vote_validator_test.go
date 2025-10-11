package services_test

import (
	"strings"
	"testing"

	"github.com/damione1/planning-poker/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestVoteValidator_ParseCustomValues(t *testing.T) {
	v := services.NewVoteValidator()

	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		// Valid cases
		{
			"t-shirt sizes",
			"XS, S, M, L, XL",
			[]string{"XS", "S", "M", "L", "XL"},
			false,
		},
		{
			"fibonacci",
			"1, 2, 3, 5, 8, 13",
			[]string{"1", "2", "3", "5", "8", "13"},
			false,
		},
		{
			"modified fibonacci",
			"0.5, 1, 2, 3, 5, 8, 13",
			[]string{"0.5", "1", "2", "3", "5", "8", "13"},
			false,
		},
		{
			"with extra spaces",
			"  XS  ,  S  ,  M  ",
			[]string{"XS", "S", "M"},
			false,
		},
		{
			"minimum 2 values",
			"Yes, No",
			[]string{"Yes", "No"},
			false,
		},
		{
			"with hyphens",
			"very-small, small, medium",
			[]string{"very-small", "small", "medium"},
			false,
		},

		// Invalid cases
		{
			"empty string",
			"",
			nil,
			true,
		},
		{
			"whitespace only",
			"   ",
			nil,
			true,
		},
		{
			"only one value",
			"XS",
			nil,
			true,
		},
		{
			"duplicate values",
			"XS, S, XS",
			nil,
			true,
		},
		{
			"too many values",
			strings.Join(make([]string, 21), ",") + strings.Repeat("X,", 21),
			nil,
			true,
		},
		{
			"value too long",
			"XXXXXXXXXXX, S",
			nil,
			true,
		},
		{
			"invalid characters",
			"XS, S@M, L",
			nil,
			true,
		},
		{
			"control characters",
			"XS\x00, S, M",
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v.ParseCustomValues(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestVoteValidator_ValidateValue(t *testing.T) {
	v := services.NewVoteValidator()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Valid
		{"numeric", "5", false},
		{"float", "0.5", false},
		{"text", "XS", false},
		{"with hyphen", "very-small", false},
		{"with underscore", "very_small", false},
		{"with space", "Very Small", false},
		{"with dot", "1.5", false},
		{"maximum length", "1234567890", false},

		// Invalid
		{"empty", "", true},
		{"too long", "12345678901", true},
		{"special char", "XS@", true},
		{"control char", "XS\n", true},
		{"null byte", "XS\x00", true},
		{"unicode emoji", "XS🚀", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateValue(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVoteValidator_ParseNumericValue(t *testing.T) {
	v := services.NewVoteValidator()

	tests := []struct {
		name      string
		value     string
		wantNum   float64
		wantValid bool
	}{
		// Valid numeric
		{"integer", "5", 5.0, true},
		{"float", "0.5", 0.5, true},
		{"zero", "0", 0.0, true},
		{"large number", "100", 100.0, true},
		{"max valid", "1000", 1000.0, true},

		// Invalid numeric
		{"negative", "-5", 0, false},
		{"too large", "1001", 0, false},
		{"text", "XS", 0, false},
		{"empty", "", 0, false},
		{"special", "?", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, valid := v.ParseNumericValue(tt.value)
			assert.Equal(t, tt.wantValid, valid)
			if tt.wantValid {
				assert.Equal(t, tt.wantNum, num)
			}
		})
	}
}

func TestVoteValidator_IsNumericValue(t *testing.T) {
	v := services.NewVoteValidator()

	tests := []struct {
		value      string
		wantNumeric bool
	}{
		{"5", true},
		{"0.5", true},
		{"100", true},
		{"XS", false},
		{"?", false},
		{"", false},
		{"-5", false},
		{"1001", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := v.IsNumericValue(tt.value)
			assert.Equal(t, tt.wantNumeric, got)
		})
	}
}

func TestVoteValidator_GetPresetTemplates(t *testing.T) {
	v := services.NewVoteValidator()

	t.Run("fibonacci", func(t *testing.T) {
		values := v.GetFibonacciValues()
		assert.NotEmpty(t, values)
		assert.Contains(t, values, "1")
		assert.Contains(t, values, "5")
		assert.Contains(t, values, "13")
	})

	t.Run("modified fibonacci", func(t *testing.T) {
		values := v.GetModifiedFibonacciValues()
		assert.NotEmpty(t, values)
		assert.Contains(t, values, "0.5")
		assert.Contains(t, values, "1")
		assert.Contains(t, values, "100")
	})

	t.Run("t-shirt", func(t *testing.T) {
		values := v.GetTShirtValues()
		assert.NotEmpty(t, values)
		assert.Contains(t, values, "XS")
		assert.Contains(t, values, "M")
		assert.Contains(t, values, "XXL")
	})
}

func TestVoteValidator_GetPresetTemplate(t *testing.T) {
	v := services.NewVoteValidator()

	tests := []struct {
		name     string
		template string
		wantErr  bool
		contains string
	}{
		{"fibonacci", services.TemplateFibonacci, false, "5"},
		{"modified fibonacci", services.TemplateModifiedFibonacci, false, "0.5"},
		{"t-shirt", services.TemplateTShirt, false, "XS"},
		{"unknown", "unknown-template", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := v.GetPresetTemplate(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, values)
				if tt.contains != "" {
					assert.Contains(t, values, tt.contains)
				}
			}
		})
	}
}

func TestVoteValidator_ValidateVoteValue(t *testing.T) {
	v := services.NewVoteValidator()

	t.Run("fibonacci pointing", func(t *testing.T) {
		tests := []struct {
			value   string
			wantErr bool
		}{
			{"1", false},
			{"5", false},
			{"13", false},
			{"?", false},
			{"☕", false},
			{"0", true},
			{"XS", true},
			{"", true},
		}

		for _, tt := range tests {
			t.Run(tt.value, func(t *testing.T) {
				err := v.ValidateVoteValue(tt.value, "fibonacci", nil)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("custom pointing", func(t *testing.T) {
		customValues := []string{"XS", "S", "M", "L", "XL"}

		tests := []struct {
			value   string
			wantErr bool
		}{
			{"XS", false},
			{"M", false},
			{"XL", false},
			{"?", false},
			{"☕", false},
			{"XXL", true},
			{"1", true},
			{"", true},
		}

		for _, tt := range tests {
			t.Run(tt.value, func(t *testing.T) {
				err := v.ValidateVoteValue(tt.value, "custom", customValues)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("custom pointing without values", func(t *testing.T) {
		err := v.ValidateVoteValue("XS", "custom", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no custom values")
	})

	t.Run("unknown pointing method", func(t *testing.T) {
		err := v.ValidateVoteValue("5", "unknown", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown pointing method")
	})
}

func TestVoteValidator_GetAvailableTemplates(t *testing.T) {
	v := services.NewVoteValidator()

	templates := v.GetAvailableTemplates()
	assert.NotEmpty(t, templates)
	assert.Len(t, templates, 3)

	// Check that all templates have required fields
	for _, tmpl := range templates {
		assert.NotEmpty(t, tmpl.ID)
		assert.NotEmpty(t, tmpl.Name)
		assert.NotEmpty(t, tmpl.Description)
		assert.NotEmpty(t, tmpl.Values)
	}
}

func TestVoteValidator_GetAvailableTemplatesMap(t *testing.T) {
	v := services.NewVoteValidator()

	templatesMap := v.GetAvailableTemplatesMap()
	assert.NotEmpty(t, templatesMap)
	assert.Contains(t, templatesMap, services.TemplateFibonacci)
	assert.Contains(t, templatesMap, services.TemplateModifiedFibonacci)
	assert.Contains(t, templatesMap, services.TemplateTShirt)
}

func TestVoteValidator_GetTemplateValuesString(t *testing.T) {
	v := services.NewVoteValidator()

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{"fibonacci", services.TemplateFibonacci, services.TemplateFibonacciValues},
		{"modified fibonacci", services.TemplateModifiedFibonacci, services.TemplateModifiedFibonacciValues},
		{"t-shirt", services.TemplateTShirt, services.TemplateTShirtValues},
		{"unknown", "unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.GetTemplateValuesString(tt.template)
			assert.Equal(t, tt.want, got)
		})
	}
}
