package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
		fmt.Fprint(out, eval(prompt))
	}
}

func eval(prompt string) string {
	if strings.HasPrefix(prompt, "exit") {
		os.Exit(0)
	}
	return fmt.Sprintf("%s: command not found\n", prompt)
}
