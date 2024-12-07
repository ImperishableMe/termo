package main

import (
	"testing"
)

func TestEval(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ls", "ls: command not found\n"},
		{"pwd", "pwd: command not found\n"},
		{"echo hello", "echo hello: command not found\n"},
	}

	for _, test := range tests {
		result := eval(test.input)
		if result != test.expected {
			t.Errorf("eval(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}
