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
	splits := strings.Split(prompt, " ")
	if len(splits) == 0 {
		return fmt.Sprintf("%s: command not found", prompt), 127, false
	}

	cmd := splits[0]
	args := splits[1:]

	if handler, ok := builtinCommands[cmd]; ok {
		return handler(args)
	}

	return RunCommand(splits)
}

func RunCommand(splits []string) (string, int, bool) {
	type Redirection struct {
		stdin  io.Reader
		stdout io.Writer
		stderr io.Writer
	}

	getRedirection := func(command string) Redirection {
		redirection := Redirection{
			stdin:  os.Stdin,
			stdout: os.Stdout,
			stderr: os.Stderr,
		}

		splits := strings.Split(command, " ")

		for i := 0; i+1 < len(splits); i++ {
			split := splits[i]
			switch split {
			case "<":
				filePath := splits[i+1]
				file, err := os.Open(filePath)
				if err != nil {
					fmt.Println("Error creating file:", err)
					return redirection
				}
				redirection.stdin = file

			case ">", "1>":
				filePath := splits[i+1]
				file, err := os.Create(filePath)
				if err != nil {
					fmt.Println("Error creating file:", err)
					return redirection
				}
				redirection.stdout = file
			case "2>":
				filePath := splits[i+1]
				file, err := os.Create(filePath)
				if err != nil {
					fmt.Println("Error creating file:", err)
					return redirection
				}
				redirection.stderr = file
			default:
				continue
			}
		}

		return redirection
	}

	ok := cmdExists(splits[0])
	if !ok {
		return fmt.Sprintf("%s: command not found", splits[0]), 127, false
	}
	cmd := exec.Command(splits[0], splits[1:]...)
	redirection := getRedirection(strings.Join(splits, " "))

	cmd.Stdin = redirection.stdin
	cmd.Stdout = redirection.stdout
	cmd.Stderr = redirection.stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", exitErr.ExitCode(), false
		}
		return fmt.Sprintf("Cmd: %s: %v", splits[0], err), 1, false
	}
	return "", 0, false
}

func cmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
