package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	if err := repl(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "repl() error = %v\n", err)
		os.Exit(1)
	}
}

func repl(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, "$ ")
		ok := scanner.Scan()
		if !ok {
			return scanner.Err()
		}
		prompt := scanner.Text()
		prompt = strings.TrimSpace(prompt)
		output, exitCode, shouldExit := eval(prompt)
		if output != "" {
			fmt.Fprintln(out, output)
		}
		if shouldExit {
			os.Exit(exitCode)
		}
	}
}

// BuiltinFunc is the signature for all builtin commands
// Returns: output string, exit code, and whether to exit the shell
type BuiltinFunc func(args []string) (output string, exitCode int, shouldExit bool)

// Redirection holds file handles for stdin/stdout/stderr redirection
type Redirection struct {
	Stdout *os.File
	Stderr *os.File
	Stdin  *os.File
}

// parseRedirections extracts redirection operators from args and returns:
// - filtered args (without redirection operators and their targets)
// - Redirection struct with opened file handles
// Caller is responsible for closing the files.
func parseRedirections(args []string) ([]string, Redirection, error) {
	var filtered []string
	redir := Redirection{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case ">", "1>":
			if i+1 >= len(args) {
				return nil, redir, fmt.Errorf("syntax error: expected filename after %s", arg)
			}
			filePath := args[i+1]
			file, err := os.Create(filePath)
			if err != nil {
				return nil, redir, fmt.Errorf("cannot open %s: %v", filePath, err)
			}
			redir.Stdout = file
			i++ // skip the filename
		case ">>", "1>>":
			if i+1 >= len(args) {
				return nil, redir, fmt.Errorf("syntax error: expected filename after %s", arg)
			}
			filePath := args[i+1]
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, redir, fmt.Errorf("cannot open %s: %v", filePath, err)
			}
			redir.Stdout = file
			i++ // skip the filename
		case "2>":
			if i+1 >= len(args) {
				return nil, redir, fmt.Errorf("syntax error: expected filename after %s", arg)
			}
			filePath := args[i+1]
			file, err := os.Create(filePath)
			if err != nil {
				return nil, redir, fmt.Errorf("cannot open %s: %v", filePath, err)
			}
			redir.Stderr = file
			i++ // skip the filename
		case "<":
			if i+1 >= len(args) {
				return nil, redir, fmt.Errorf("syntax error: expected filename after %s", arg)
			}
			filePath := args[i+1]
			file, err := os.Open(filePath)
			if err != nil {
				return nil, redir, fmt.Errorf("cannot open %s: %v", filePath, err)
			}
			redir.Stdin = file
			i++ // skip the filename
		default:
			filtered = append(filtered, arg)
		}
	}

	return filtered, redir, nil
}

// closeRedirections closes all open file handles in a Redirection struct
func closeRedirections(redir Redirection) {
	if redir.Stdout != nil {
		redir.Stdout.Close()
	}
	if redir.Stderr != nil {
		redir.Stderr.Close()
	}
	if redir.Stdin != nil {
		redir.Stdin.Close()
	}
}

// parseArgs splits a command line into arguments, handling single-quoted strings.
// Single quotes preserve literal text and are stripped from the output.
func parseArgs(input string) []string {
	var args []string
	var current strings.Builder
	inSingleQuote := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case ch == '\'' && !inSingleQuote:
			inSingleQuote = true
		case ch == '\'' && inSingleQuote:
			inSingleQuote = false
		case ch == ' ' && !inSingleQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

var builtinCommands map[string]BuiltinFunc

func init() {
	builtinCommands = map[string]BuiltinFunc{
		"exit": builtinExit,
		"echo": builtinEcho,
		"type": builtinType,
	}
}

func builtinExit(args []string) (string, int, bool) {
	code := 0
	if len(args) > 0 {
		codeInt, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Sprintf("exit: %s: numeric argument required", args[0]), 2, true
		}
		code = int(codeInt)
	}
	return "", code, true
}

func builtinEcho(args []string) (string, int, bool) {
	return strings.Join(args, " "), 0, false
}

func builtinType(args []string) (string, int, bool) {
	if len(args) < 1 {
		return "type: usage: type name", 1, false
	}

	if _, ok := builtinCommands[args[0]]; ok {
		return fmt.Sprintf("%s is a shell builtin", args[0]), 0, false
	}

	if path, err := exec.LookPath(args[0]); err == nil {
		return fmt.Sprintf("%s is %s", args[0], path), 0, false
	}
	return fmt.Sprintf("%s: not found", args[0]), 1, false
}

func eval(prompt string) (output string, exitCode int, shouldExit bool) {
	splits := parseArgs(prompt)
	if len(splits) == 0 {
		return fmt.Sprintf("%s: command not found", prompt), 127, false
	}

	cmd := splits[0]
	args := splits[1:]

	// Parse redirections from args
	filteredArgs, redir, err := parseRedirections(args)
	defer closeRedirections(redir)

	if err != nil {
		return err.Error(), 1, false
	}

	if handler, ok := builtinCommands[cmd]; ok {
		output, exitCode, shouldExit = handler(filteredArgs)
		// If stdout is redirected, write output to file instead of returning
		if redir.Stdout != nil && output != "" {
			fmt.Fprintln(redir.Stdout, output)
			return "", exitCode, shouldExit
		}
		return output, exitCode, shouldExit
	}

	return runCommandWithRedirection(cmd, filteredArgs, redir)
}

// runCommandWithRedirection runs an external command with the given args and redirection
func runCommandWithRedirection(cmdName string, args []string, redir Redirection) (string, int, bool) {
	if _, err := exec.LookPath(cmdName); err != nil {
		return fmt.Sprintf("%s: command not found", cmdName), 127, false
	}

	cmd := exec.Command(cmdName, args...)

	// Set up I/O redirection
	if redir.Stdin != nil {
		cmd.Stdin = redir.Stdin
	} else {
		cmd.Stdin = os.Stdin
	}

	if redir.Stdout != nil {
		cmd.Stdout = redir.Stdout
	} else {
		cmd.Stdout = os.Stdout
	}

	if redir.Stderr != nil {
		cmd.Stderr = redir.Stderr
	} else {
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", exitErr.ExitCode(), false
		}
		return fmt.Sprintf("Cmd: %s: %v", cmdName, err), 1, false
	}
	return "", 0, false
}
