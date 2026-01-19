package main

import (
	"testing"
)

func TestEval(t *testing.T) {
	tests := []struct {
		input        string
		wantOutput   string
		wantExitCode int
		wantExit     bool
	}{
		{"echo hello", "hello", 0, false},
		{"echo hello world", "hello world", 0, false},
		{"nonexistent_cmd", "nonexistent_cmd: command not found", 127, false},
	}

	for _, test := range tests {
		output, exitCode, shouldExit := eval(test.input)
		if output != test.wantOutput {
			t.Errorf("eval(%q) output = %q; want %q", test.input, output, test.wantOutput)
		}
		if exitCode != test.wantExitCode {
			t.Errorf("eval(%q) exitCode = %d; want %d", test.input, exitCode, test.wantExitCode)
		}
		if shouldExit != test.wantExit {
			t.Errorf("eval(%q) shouldExit = %v; want %v", test.input, shouldExit, test.wantExit)
		}
	}
}

func TestBuiltinExit(t *testing.T) {
	output, exitCode, shouldExit := builtinExit([]string{"42"})
	if output != "" {
		t.Errorf("builtinExit output = %q; want empty", output)
	}
	if exitCode != 42 {
		t.Errorf("builtinExit exitCode = %d; want 42", exitCode)
	}
	if !shouldExit {
		t.Errorf("builtinExit shouldExit = false; want true")
	}
}

func TestBuiltinType(t *testing.T) {
	tests := []struct {
		args       []string
		wantOutput string
	}{
		{[]string{"echo"}, "echo is a shell builtin"},
		{[]string{"exit"}, "exit is a shell builtin"},
		{[]string{"type"}, "type is a shell builtin"},
	}

	for _, test := range tests {
		output, exitCode, _ := builtinType(test.args)
		if output != test.wantOutput {
			t.Errorf("builtinType(%v) output = %q; want %q", test.args, output, test.wantOutput)
		}
		if exitCode != 0 {
			t.Errorf("builtinType(%v) exitCode = %d; want 0", test.args, exitCode)
		}
	}
}
