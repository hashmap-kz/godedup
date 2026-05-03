package load

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoadDirectoryRecursively(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.go"), `package sample
func One() int {
	a := 1
	b := 2
	return a + b
}`)
	writeFile(t, filepath.Join(dir, "nested", "b.go"), `package sample
func Two() int {
	a := 1
	b := 2
	return a + b
}`)

	result, err := Load([]string{dir}, false)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(result.Funcs) != 2 {
		t.Fatalf("len(Funcs) = %d, want 2", len(result.Funcs))
	}
	if result.Fset == nil {
		t.Fatal("Fset is nil")
	}
}

func TestLoadExcludesTests(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.go"), `package sample
func One() int {
	a := 1
	b := 2
	return a + b
}`)
	writeFile(t, filepath.Join(dir, "a_test.go"), `package sample
func TestOne() int {
	a := 1
	b := 2
	return a + b
}`)

	result, err := Load([]string{dir}, true)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(result.Funcs) != 1 {
		t.Fatalf("len(Funcs) = %d, want 1", len(result.Funcs))
	}
	if got := result.Funcs[0].Name; got != "sample.One" {
		t.Fatalf("loaded function = %q, want sample.One", got)
	}
}

func TestLoadIncludesTestsWhenNotExcluded(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.go"), `package sample
func One() int {
	a := 1
	b := 2
	return a + b
}`)
	writeFile(t, filepath.Join(dir, "a_test.go"), `package sample
func TestOne() int {
	a := 1
	b := 2
	return a + b
}`)

	result, err := Load([]string{dir}, false)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(result.Funcs) != 2 {
		t.Fatalf("len(Funcs) = %d, want 2", len(result.Funcs))
	}
}

func TestLoadSkipsShortFunctionsAndUnparseableFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "short.go"), `package sample
func Short() int {
	return 1
}`)
	writeFile(t, filepath.Join(dir, "bad.go"), `package sample
func Broken(`)
	writeFile(t, filepath.Join(dir, "good.go"), `package sample
func Good() int {
	a := 1
	b := 2
	return a + b
}`)

	result, err := Load([]string{dir}, false)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(result.Funcs) != 1 {
		t.Fatalf("len(Funcs) = %d, want 1", len(result.Funcs))
	}
	if got := result.Funcs[0].Name; got != "sample.Good" {
		t.Fatalf("loaded function = %q, want sample.Good", got)
	}
}

func TestLoadSkipsHiddenAndVendorDirectories(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "good.go"), `package sample
func Good() int {
	a := 1
	b := 2
	return a + b
}`)
	writeFile(t, filepath.Join(dir, ".hidden", "hidden.go"), `package hidden
func Hidden() int {
	a := 1
	b := 2
	return a + b
}`)
	writeFile(t, filepath.Join(dir, "vendor", "vendored.go"), `package vendor
func Vendored() int {
	a := 1
	b := 2
	return a + b
}`)

	result, err := Load([]string{dir}, false)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(result.Funcs) != 1 {
		t.Fatalf("len(Funcs) = %d, want 1", len(result.Funcs))
	}
	if got := result.Funcs[0].Name; got != "sample.Good" {
		t.Fatalf("loaded function = %q, want sample.Good", got)
	}
}

func TestLoadSingleFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "single.go")
	writeFile(t, file, `package sample
func Single() int {
	a := 1
	b := 2
	return a + b
}`)

	result, err := Load([]string{file}, false)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(result.Funcs) != 1 {
		t.Fatalf("len(Funcs) = %d, want 1", len(result.Funcs))
	}
}
