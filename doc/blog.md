# Building a Terminal Shell in Go

This blog post walks through building a POSIX-compliant shell in Go, following the CodeCrafters "Build Your Own Shell" challenge.

## Introduction

A shell is a command-line interpreter that provides a user interface for accessing operating system services. In this series, we'll build one from scratch in Go.

## Part 1: The REPL

The foundation of any shell is the Read-Eval-Print Loop (REPL):

```go
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
```

This loop:
1. Prints a prompt (`$ `)
2. Reads a line of input
3. Evaluates the command, receiving output, exit code, and whether to exit
4. Prints the result (if any)
5. Exits the shell if the command requested it (e.g., `exit`)

## Part 2: Builtin Commands

Some commands are built into the shell itself rather than being external programs:

- **exit**: Terminates the shell with an optional exit code
- **echo**: Prints arguments to stdout
- **type**: Shows whether a command is a builtin or external program

We use a function-based registry with a consistent handler signature:

```go
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
```

Each builtin is a separate function with the same signature, making them easy to test and extend:

```go
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
```

## Part 3: The `type` Command and PATH Resolution

The `type` command shows whether a command is a builtin or an external program. For external programs, it uses Go's `exec.LookPath()` to search the PATH:

```go
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
```

Using `exec.LookPath()` is cleaner than manually iterating through PATH directories and handles edge cases like checking executable permissions.

## Part 4: Running External Commands

For commands that aren't builtins, we use Go's `os/exec` package. The `eval()` function dispatches to builtins or external commands:

```go
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
```

The `RunCommand` function executes external programs and returns the same tuple as builtins:

```go
func RunCommand(splits []string) (string, int, bool) {
    ok := cmdExists(splits[0])
    if !ok {
        return fmt.Sprintf("%s: command not found", splits[0]), 127, false
    }

    cmd := exec.Command(splits[0], splits[1:]...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err := cmd.Run()
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            return "", exitErr.ExitCode(), false
        }
        return fmt.Sprintf("Cmd: %s: %v", splits[0], err), 1, false
    }
    return "", 0, false
}
```

## Part 5: I/O Redirection

Shells support redirecting input and output to files:

- `>` or `1>`: Redirect stdout to a file
- `2>`: Redirect stderr to a file
- `<`: Read stdin from a file

For example:
```bash
echo hello > output.txt      # writes "hello" to output.txt
cat nonexistent 2> err.txt   # writes error message to err.txt
cat < input.txt              # reads from input.txt
```

### Testing CLI Tools: The txtar Approach

Testing a shell presents a challenge: how do you verify that `echo hello > output.txt` actually creates a file with the right content? Traditional Go unit tests quickly become awkward—you end up with verbose setup code, temporary directories, subprocess management, and string comparisons that obscure what you're actually testing.

Russ Cox faced this problem when testing the Go toolchain itself. His solution was `testscript`, a package that treats tests as scripts rather than code. The insight: **tests for CLI tools should look like CLI usage**.

Instead of:

```go
func TestRedirection(t *testing.T) {
    dir := t.TempDir()
    outFile := filepath.Join(dir, "output.txt")
    cmd := exec.Command(binary)
    cmd.Stdin = strings.NewReader("echo hello > " + outFile + "\nexit\n")
    // ... 20 more lines of setup and assertions
}
```

You write a `.txtar` file that reads like documentation:

```
# Test basic stdout redirection with >
stdin commands.txt
exec myshell

cmp output.txt expected.txt

-- commands.txt --
echo hello > output.txt
exit

-- expected.txt --
hello
```

This is the same format used to test `go build`, `go mod`, and other Go tools. The philosophy: if your test doesn't look like what a user would actually type, you're testing the wrong abstraction.

The txtar format bundles everything together—the commands to run, the input files, the expected outputs. Each test runs in an isolated sandbox, so there's no cleanup code. Adding a new test case means adding a new `.txtar` file, not modifying Go code.

For a shell implementation, this is particularly fitting. We're building a tool that interprets text commands and manipulates files. Our tests should be text commands that manipulate files.

### Implementation

The key insight is that redirection must apply to both builtin commands (like `echo`) and external commands (like `ls`). We parse redirections early in `eval()` and apply them uniformly.

First, define a `Redirection` struct and parser:

```go
type Redirection struct {
    Stdout *os.File
    Stderr *os.File
    Stdin  *os.File
}

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
        // ... similar for 2> and <
        default:
            filtered = append(filtered, arg)
        }
    }
    return filtered, redir, nil
}
```

Then modify `eval()` to parse redirections and apply them:

```go
func eval(prompt string) (output string, exitCode int, shouldExit bool) {
    splits := strings.Split(prompt, " ")
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
        // If stdout is redirected, write output to file instead
        if redir.Stdout != nil && output != "" {
            fmt.Fprintln(redir.Stdout, output)
            return "", exitCode, shouldExit
        }
        return output, exitCode, shouldExit
    }

    return runCommandWithRedirection(cmd, filteredArgs, redir)
}
```

For external commands, we set up the `exec.Command` with the redirected file handles:

```go
func runCommandWithRedirection(cmdName string, args []string, redir Redirection) (string, int, bool) {
    cmd := exec.Command(cmdName, args...)

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
    // ... handle errors and return
}
```

The key points:
1. **Parse early**: Extract redirection operators before dispatching to handlers
2. **Filter args**: Remove `>`, `1>`, `2>`, `<` and their targets from the argument list
3. **Apply uniformly**: Both builtins and external commands use the same redirection logic
4. **Clean up**: Use `defer` to close file handles

---

*This blog post is a work in progress. More sections will be added as we implement additional features.*
