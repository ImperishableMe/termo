package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

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
		fmt.Fprintln(out, eval(prompt))
	}
}

var builtinCommands = map[string]bool{
	"exit": true,
	"echo": true,
	"type": true,
}

func eval(prompt string) string {
	splits := strings.Split(prompt, " ")
	if len(splits) == 0 {
		return fmt.Sprintf("%s: command not found", prompt)
	}

	switch splits[0] {
	case "exit":
		code := "0"
		if len(splits) > 1 {
			code = splits[1]
		}
		codeInt, err := strconv.ParseInt(code, 10, 64)
		if err != nil {
			return fmt.Sprintf("exit: %s: numeric argument required", code)
		}
		os.Exit(int(codeInt))
	case "echo":
		return strings.Join(splits[1:], " ")
	case "type":
		return typeF(splits[1:])
	default:
		ok := cmdExists(splits[0])
		if !ok {
			return fmt.Sprintf("%s: command not found", splits[0])
		}
		cmd := exec.Command(splits[0], splits[1:]...)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Sprintf("%s: %v", splits[0], err)
		}
		return string(output)
	}
	return ""
}

func typeF(splits []string) string {
	if len(splits) < 1 {
		return fmt.Sprintf("type: usage: type name")
	}

	if _, ok := builtinCommands[splits[0]]; ok {
		return fmt.Sprintf("%s is a shell builtin", splits[0])
	}
	path := os.Getenv("PATH")
	commandPaths := strings.Split(path, string(os.PathListSeparator))

	for _, commandPath := range commandPaths {
		filePath := filepath.Join(commandPath, splits[0])
		if fileExists(filePath) {
			return fmt.Sprintf("%s is %s", splits[0], filePath)
		}
	}
	return fmt.Sprintf("%s: not found", splits[0])
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func cmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
