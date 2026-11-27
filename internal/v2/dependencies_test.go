package v2_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDependencyRules(t *testing.T) {
	t.Parallel()
	module := modulePath(t)

	rules := []struct {
		layer     string
		forbidden []string
	}{
		{
			layer: "handlers",
			forbidden: []string{
				module + "/internal/v2/services",
				module + "/internal/v2/repositories",
			},
		},
		{
			layer: "services",
			forbidden: []string{
				module + "/internal/v2/repositories",
				module + "/internal/v2/gen",
				module + "/pkg/connector/asyncapi",
			},
		},
	}

	for _, rule := range rules {
		t.Run(rule.layer, func(t *testing.T) {
			t.Parallel()
			checkImports(t, rule.layer, rule.forbidden)
		})
	}
}

func checkImports(t *testing.T, layer string, forbidden []string) {
	t.Helper()

	err := filepath.WalkDir(layer, func(
		path string,
		entry fs.DirEntry,
		err error,
	) error {
		if err != nil {
			return err
		}

		if entry.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return nil
		}

		checkFileImports(t, path, layer, forbidden)

		return nil
	})
	if err != nil {
		t.Fatalf("scan %s: %v", layer, err)
	}
}

func checkFileImports(t *testing.T, path, layer string, forbidden []string) {
	t.Helper()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Errorf("parse %s: %v", path, err)

		return
	}

	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Errorf("%s: invalid import: %v", fset.Position(spec.Pos()), err)

			continue
		}

		for _, prefix := range forbidden {
			if importPath == prefix ||
				strings.HasPrefix(importPath, prefix+"/") {
				t.Errorf(
					"%s: %s must not import %q",
					fset.Position(spec.Pos()), layer, importPath,
				)
			}
		}
	}
}

func modulePath(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	for {
		// The search is limited to go.mod in ancestors of the test directory.
		data, err := os.ReadFile(filepath.Join(dir, "go.mod")) // #nosec G304
		if err == nil {
			for line := range strings.SplitSeq(string(data), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 && fields[0] == "module" {
					return strings.Trim(fields[1], `"`)
				}
			}

			t.Fatal("go.mod has no module directive")
		}

		if !os.IsNotExist(err) {
			t.Fatalf("read go.mod: %v", err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}

		dir = parent
	}
}
