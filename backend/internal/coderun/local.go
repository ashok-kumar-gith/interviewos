package coderun

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LocalExecutor runs code on the host using locally-installed toolchains
// (python3, node, go, gcc/g++, javac/java). It is the default executor because
// the public Piston API became whitelist-only (2026-02). Each run executes in a
// fresh temp dir and is bounded by the context deadline the service sets.
//
// NOTE: this is a developer-convenience runner, not a hardened sandbox. It runs
// untrusted code with the server's privileges, so it is intended for local/dev
// use; a production deployment should point the Executor at a sandboxed runner
// (gVisor/Firecracker/containerized Piston) instead.
type LocalExecutor struct {
	// toolOverride lets tests stub the resolved interpreter/compiler paths.
	lookPath func(string) (string, error)
}

// NewLocalExecutor constructs a LocalExecutor using the host PATH.
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{lookPath: exec.LookPath}
}

// Execute implements Executor by writing the source to a temp dir and invoking
// the right local toolchain. Transport-style failures (missing toolchain) return
// ErrExecutorUnavailable; a program that ran but exited non-zero is Ran=true.
func (l *LocalExecutor) Execute(ctx context.Context, in RunInput) (RunOutput, error) {
	dir, err := os.MkdirTemp("", "coderun-*")
	if err != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	defer os.RemoveAll(dir)

	out := RunOutput{Language: in.Language, Version: "local"}

	// The service passes the resolved Piston runtime name (e.g. "c++", "node");
	// normalize back to our canonical keys so the switch handles either form.
	switch normalizeLang(in.Language) {
	case "python":
		return l.interpret(ctx, dir, in, out, "python3", "main.py", []string{"main.py"})
	case "javascript":
		return l.interpret(ctx, dir, in, out, "node", "main.js", []string{"main.js"})
	case "typescript":
		// Run TS via node by transpiling-on-run when ts-node/tsx exist; otherwise
		// fall back to executing as JS (most interview snippets are valid JS).
		if bin, err := l.lookPath("tsx"); err == nil {
			return l.runWith(ctx, dir, in, out, "main.ts", bin, []string{bin, "main.ts"})
		}
		return l.interpret(ctx, dir, in, out, "node", "main.js", []string{"main.js"})
	case "go":
		return l.runGo(ctx, dir, in, out)
	case "c":
		return l.compileC(ctx, dir, in, out, "cc")
	case "cpp":
		return l.compileC(ctx, dir, in, out, "c++")
	case "java":
		return l.runJava(ctx, dir, in, out)
	default:
		return RunOutput{}, ErrUnsupportedLanguage
	}
}

// interpret writes the source and runs an interpreter directly.
func (l *LocalExecutor) interpret(ctx context.Context, dir string, in RunInput, out RunOutput, tool, file string, args []string) (RunOutput, error) {
	bin, err := l.lookPath(tool)
	if err != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	if werr := os.WriteFile(filepath.Join(dir, file), []byte(in.Source), 0o600); werr != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	return l.runWith(ctx, dir, in, out, file, bin, append([]string{bin}, args...))
}

// runGo runs a single-file Go program via `go run`.
func (l *LocalExecutor) runGo(ctx context.Context, dir string, in RunInput, out RunOutput) (RunOutput, error) {
	bin, err := l.lookPath("go")
	if err != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	if werr := os.WriteFile(filepath.Join(dir, "main.go"), []byte(in.Source), 0o600); werr != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	// GOCACHE/GOPATH inside the temp dir keep runs hermetic and writable.
	env := append(os.Environ(), "GOCACHE="+filepath.Join(dir, ".gocache"), "GOFLAGS=-mod=mod")
	return l.spawn(ctx, dir, in.Stdin, out, env, bin, "run", "main.go")
}

// compileC compiles a C/C++ source then runs the binary.
func (l *LocalExecutor) compileC(ctx context.Context, dir string, in RunInput, out RunOutput, tool string) (RunOutput, error) {
	bin, err := l.lookPath(tool)
	if err != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	src := "main.c"
	if tool == "c++" {
		src = "main.cpp"
	}
	if werr := os.WriteFile(filepath.Join(dir, src), []byte(in.Source), 0o600); werr != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	exe := filepath.Join(dir, "a.out")
	// Compile step: a non-zero compile is a Ran=true result with stderr (the
	// compiler errors), mirroring how the user would see a failed build.
	cstdout, cstderr, code, cerr := runCmd(ctx, dir, "", os.Environ(), bin, src, "-o", exe)
	if cerr != nil {
		return RunOutput{}, cerr
	}
	if code != 0 {
		out.Ran = true
		out.Stdout = cstdout
		out.Stderr = cstderr
		out.ExitCode = code
		out.Message = "compilation failed"
		return out, nil
	}
	return l.spawn(ctx, dir, in.Stdin, out, os.Environ(), exe)
}

// runJava compiles Main.java then runs it.
func (l *LocalExecutor) runJava(ctx context.Context, dir string, in RunInput, out RunOutput) (RunOutput, error) {
	javac, err := l.lookPath("javac")
	if err != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	java, err := l.lookPath("java")
	if err != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	if werr := os.WriteFile(filepath.Join(dir, "Main.java"), []byte(in.Source), 0o600); werr != nil {
		return RunOutput{}, ErrExecutorUnavailable
	}
	cstdout, cstderr, code, cerr := runCmd(ctx, dir, "", os.Environ(), javac, "Main.java")
	if cerr != nil {
		return RunOutput{}, cerr
	}
	if code != 0 {
		out.Ran = true
		out.Stdout = cstdout
		out.Stderr = cstderr
		out.ExitCode = code
		out.Message = "compilation failed"
		return out, nil
	}
	return l.spawn(ctx, dir, in.Stdin, out, os.Environ(), java, "-cp", dir, "Main")
}

// runWith runs an already-written interpreted file.
func (l *LocalExecutor) runWith(ctx context.Context, dir string, in RunInput, out RunOutput, file, bin string, argv []string) (RunOutput, error) {
	_ = file
	return l.spawn(ctx, dir, in.Stdin, out, os.Environ(), argv[0], argv[1:]...)
}

// spawn runs argv in dir, feeding stdin, and fills the RunOutput. A non-zero
// exit is a successful Ran=true result, not an error.
func (l *LocalExecutor) spawn(ctx context.Context, dir, stdin string, out RunOutput, env []string, name string, args ...string) (RunOutput, error) {
	stdout, stderr, code, err := runCmd(ctx, dir, stdin, env, name, args...)
	if err != nil {
		return RunOutput{}, err
	}
	out.Ran = true
	out.Stdout = stdout
	out.Stderr = stderr
	out.ExitCode = code
	if ctx.Err() == context.DeadlineExceeded {
		out.Message = "execution timed out"
	}
	return out, nil
}

// runCmd executes a command bounded by ctx, returning stdout, stderr, exit code.
// It returns an error only for spawn failures, not for non-zero exits or
// timeouts (a timeout kills the process and surfaces via a non-zero code).
func runCmd(ctx context.Context, dir, stdin string, env []string, name string, args ...string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()

	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			exitCode = 124 // conventional timeout code
		} else {
			// Couldn't start the process at all (missing binary, etc.).
			return "", "", 0, ErrExecutorUnavailable
		}
	}
	return truncate(so.String()), truncate(se.String()), exitCode, nil
}

// normalizeLang maps both the public language id and the Piston runtime name to
// the canonical key used by Execute's switch (e.g. "c++"→"cpp", "node"→"javascript").
func normalizeLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "c++", "cpp", "g++":
		return "cpp"
	case "node", "nodejs", "javascript", "js":
		return "javascript"
	case "ts", "typescript":
		return "typescript"
	case "py", "python", "python3":
		return "python"
	case "golang", "go":
		return "go"
	default:
		return strings.ToLower(strings.TrimSpace(lang))
	}
}

// truncate caps captured output so a runaway program can't return megabytes.
func truncate(s string) string {
	const max = 64 * 1024
	if len(s) > max {
		return s[:max] + "\n…[output truncated]"
	}
	return s
}
