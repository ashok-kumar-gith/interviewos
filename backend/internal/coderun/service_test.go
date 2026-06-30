package coderun

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeExecutor is a test double for the Executor port. It records the input it
// received and returns a canned output/error.
type fakeExecutor struct {
	out    RunOutput
	err    error
	gotIn  RunInput
	called bool
}

func (f *fakeExecutor) Execute(_ context.Context, in RunInput) (RunOutput, error) {
	f.called = true
	f.gotIn = in
	return f.out, f.err
}

func TestRun_Success(t *testing.T) {
	fake := &fakeExecutor{out: RunOutput{
		Stdout:   "2\n",
		Stderr:   "",
		ExitCode: 0,
		Version:  "3.12.0",
		Ran:      true,
	}}
	svc := NewService(ServiceConfig{Executor: fake})

	out, err := svc.Run(context.Background(), "python", "print(1+1)", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Ran {
		t.Errorf("expected Ran=true")
	}
	if out.Stdout != "2\n" {
		t.Errorf("stdout = %q, want %q", out.Stdout, "2\n")
	}
	if out.ExitCode != 0 {
		t.Errorf("exit_code = %d, want 0", out.ExitCode)
	}
	if out.Language != "python" {
		t.Errorf("language = %q, want python", out.Language)
	}
	// The service must translate the public language to the Piston runtime name.
	if fake.gotIn.Language != "python" {
		t.Errorf("executor language = %q, want python", fake.gotIn.Language)
	}
}

func TestRun_CppMapsToPistonRuntimeName(t *testing.T) {
	fake := &fakeExecutor{out: RunOutput{Ran: true, Version: "10.2.0"}}
	svc := NewService(ServiceConfig{Executor: fake})

	out, err := svc.Run(context.Background(), "CPP", "int main(){}", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Internal executor sees the Piston runtime name "c++"...
	if fake.gotIn.Language != "c++" {
		t.Errorf("executor language = %q, want c++", fake.gotIn.Language)
	}
	// ...but the caller-facing language stays the public identifier "cpp".
	if out.Language != "cpp" {
		t.Errorf("response language = %q, want cpp", out.Language)
	}
}

func TestRun_UnsupportedLanguage(t *testing.T) {
	fake := &fakeExecutor{}
	svc := NewService(ServiceConfig{Executor: fake})

	_, err := svc.Run(context.Background(), "rust", "fn main(){}", "")
	if !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("err = %v, want ErrUnsupportedLanguage", err)
	}
	if fake.called {
		t.Errorf("executor should not be called for an unsupported language")
	}
}

func TestRun_RuntimeError(t *testing.T) {
	// A program that compiled-and-ran but exited non-zero is a SUCCESSFUL run:
	// nil error, Ran=true, stderr populated, non-zero exit code.
	fake := &fakeExecutor{out: RunOutput{
		Stdout:   "",
		Stderr:   "Traceback (most recent call last):\nZeroDivisionError: division by zero\n",
		ExitCode: 1,
		Version:  "3.12.0",
		Ran:      true,
	}}
	svc := NewService(ServiceConfig{Executor: fake})

	out, err := svc.Run(context.Background(), "python", "print(1/0)", "")
	if err != nil {
		t.Fatalf("runtime error should not be a service error: %v", err)
	}
	if !out.Ran {
		t.Errorf("expected Ran=true for a runtime error")
	}
	if out.ExitCode != 1 {
		t.Errorf("exit_code = %d, want 1", out.ExitCode)
	}
	if !strings.Contains(out.Stderr, "ZeroDivisionError") {
		t.Errorf("stderr = %q, want it to contain ZeroDivisionError", out.Stderr)
	}
}

func TestRun_EmptySource(t *testing.T) {
	svc := NewService(ServiceConfig{Executor: &fakeExecutor{}})
	_, err := svc.Run(context.Background(), "python", "   \n  ", "")
	if !errors.Is(err, ErrEmptySource) {
		t.Fatalf("err = %v, want ErrEmptySource", err)
	}
}

func TestRun_SourceTooLarge(t *testing.T) {
	svc := NewService(ServiceConfig{Executor: &fakeExecutor{}})
	big := strings.Repeat("x", MaxSourceBytes+1)
	_, err := svc.Run(context.Background(), "python", big, "")
	if !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("err = %v, want ErrSourceTooLarge", err)
	}
}

func TestRun_StdinTooLarge(t *testing.T) {
	svc := NewService(ServiceConfig{Executor: &fakeExecutor{}})
	big := strings.Repeat("y", MaxStdinBytes+1)
	_, err := svc.Run(context.Background(), "python", "print(1)", big)
	if !errors.Is(err, ErrStdinTooLarge) {
		t.Fatalf("err = %v, want ErrStdinTooLarge", err)
	}
}

func TestRun_ExecutorUnavailable(t *testing.T) {
	fake := &fakeExecutor{err: ErrExecutorUnavailable}
	svc := NewService(ServiceConfig{Executor: fake})

	_, err := svc.Run(context.Background(), "python", "print(1)", "")
	if !errors.Is(err, ErrExecutorUnavailable) {
		t.Fatalf("err = %v, want ErrExecutorUnavailable", err)
	}
}

func TestNormalizePistonResponse_CompileFailureSurfacesStderr(t *testing.T) {
	code := 1
	pr := pistonExecuteResponse{
		Language: "c++",
		Version:  "10.2.0",
		Compile:  &pistonStage{Stderr: "error: expected ';'", Code: &code},
		Run:      pistonStage{},
	}
	out := normalizePistonResponse(pr, "cpp")
	if !out.Ran {
		t.Errorf("expected Ran=true")
	}
	if out.ExitCode != 1 {
		t.Errorf("exit_code = %d, want 1", out.ExitCode)
	}
	if !strings.Contains(out.Stderr, "expected ';'") {
		t.Errorf("stderr = %q, want compile diagnostics", out.Stderr)
	}
	if out.Message != "compilation failed" {
		t.Errorf("message = %q, want 'compilation failed'", out.Message)
	}
}
