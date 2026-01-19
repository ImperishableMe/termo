# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a POSIX-compliant shell implementation built as part of the CodeCrafters "Build Your Own Shell" challenge. The shell interprets commands, runs external programs, and supports builtin commands (exit, echo, type).

## Build and Run Commands

```bash
# Build and run the shell
./your_program.sh

# Build only
go build -o /tmp/shell-target cmd/myshell/*.go

# Run tests
go test ./cmd/myshell/...

# Run a single test
go test ./cmd/myshell/... -run TestEval
```

## Architecture

The shell is implemented in a single file (`cmd/myshell/main.go`) with:

- **REPL loop** (`repl`): Reads input, evaluates, prints output
- **Eval function** (`eval`): Dispatches commands to builtins or external execution
- **Builtin commands**: `exit`, `echo`, `type` - defined in `builtinCommands` map
- **External command execution** (`RunCommand`): Uses `exec.Command` with I/O redirection support (`>`, `1>`, `2>`, `<`)
- **PATH resolution** (`typeF`): Searches PATH directories for executables

## CodeCrafters Integration

- Submit solutions via `git push origin master`
- Do not edit `go.mod` - CodeCrafters relies on it
- Tests run remotely on CodeCrafters infrastructure

## Blog Post

We are writing a blog post about how to build a terminal shell in Go following the tasks in this codebase. Create and maintain a `doc/blog.md` file as we progress through the project. When we are making incremental changes, add along instead of overwriting, only overwrite when we make a mistake earlier.
