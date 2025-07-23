package vi_test

import (
	"testing"

	"fortio.org/gvi/vi"
)

func TestFilterSpecialChars(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, World!", "Hello, World!"},
		{"Hello\x7f, Wor\x03ld!", "Hello, World!"},
		{"\x01Hello", "Hello"},
		{"Smiley 😊 :) \x02", "Smiley 😊 :) "},
		{"Smiley 😊 :) \x02", "Smiley 😊 :) "},
		{"\x01\x02\x03", ""},
		{"A\x00B", "AB"},
	}
	for _, test := range tests {
		output := vi.FilterSpecialChars(test.input)
		if output != test.expected {
			t.Errorf("FilterSpecialChars(%q) = %q; expected %q", test.input, output, test.expected)
		}
	}
}
