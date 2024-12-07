package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
	default:
		return fmt.Sprintf("%s: command not found", prompt)
	}
	return ""
}
