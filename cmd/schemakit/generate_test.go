package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCheckSchemaDrift(t *testing.T) {
	dir := t.TempDir()
	cmd := &cobra.Command{}

	match := filepath.Join(dir, "match.json")
	if err := os.WriteFile(match, []byte("{\"a\":1}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	drift := filepath.Join(dir, "drift.json")
	if err := os.WriteFile(drift, []byte("{\"a\":2}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.json")

	// Matching content (ignoring trailing newline) is not drift.
	if err := checkSchemaDrift(cmd, match, "{\"a\":1}"); err != nil {
		t.Errorf("matching content should not report drift: %v", err)
	}

	// Differing content is drift.
	if err := checkSchemaDrift(cmd, drift, "{\"a\":1}"); err == nil {
		t.Error("differing content should report drift")
	}

	// A missing file is drift (must be generated first).
	if err := checkSchemaDrift(cmd, missing, "{\"a\":1}"); err == nil {
		t.Error("missing file should report drift")
	}
}

// parseProgram parses a rendered generator program, failing the test if it
// is not valid Go.
func parseProgram(t *testing.T, src []byte) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "gen.go", src, 0)
	if err != nil {
		t.Fatalf("rendered program is not valid Go: %v\n%s", err, src)
	}
	return f
}

// assignedString returns the string literal assigned to schema.<field>, or
// ok=false when the program has no such assignment.
func assignedString(t *testing.T, f *ast.File, field string) (string, bool) {
	t.Helper()
	var (
		val   string
		found bool
	)
	ast.Inspect(f, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 {
			return true
		}
		sel, ok := as.Lhs[0].(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != field {
			return true
		}
		var lit *ast.BasicLit
		switch rhs := as.Rhs[0].(type) {
		case *ast.BasicLit:
			lit = rhs
		case *ast.CallExpr: // schema.ID = jsonschema.ID("...")
			if len(rhs.Args) == 1 {
				lit, _ = rhs.Args[0].(*ast.BasicLit)
			}
		}
		if lit == nil {
			return true
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			t.Fatalf("unquote %s: %v", lit.Value, err)
		}
		val, found = s, true
		return false
	})
	return val, found
}

func TestRenderProgramQuotesMetadata(t *testing.T) {
	// Values chosen to break naive interpolation: quotes, backslashes,
	// newlines, backticks, and text that looks like Go code.
	hostile := "a \"quoted\" \\ back\\slash\nnew`line` \"; os.Exit(1); //"
	src, err := renderProgram(genProgram{
		Package: "example.com/m/pkg", Type: "Thing", Indent: true,
		ID: "https://example.com/" + hostile, Title: hostile, Description: hostile,
	})
	if err != nil {
		t.Fatal(err)
	}
	f := parseProgram(t, src)

	for _, field := range []string{"Title", "Description"} {
		got, ok := assignedString(t, f, field)
		if !ok || got != hostile {
			t.Errorf("%s = %q (found=%v), want exact round trip of %q", field, got, ok, hostile)
		}
	}
	if got, ok := assignedString(t, f, "ID"); !ok || got != "https://example.com/"+hostile {
		t.Errorf("ID = %q (found=%v)", got, ok)
	}

	// Structural injection check: hostile values must add no statements, so
	// the number of real os.Exit calls matches a program with harmless values.
	harmless, err := renderProgram(genProgram{
		Package: "example.com/m/pkg", Type: "Thing", Indent: true,
		ID: "id", Title: "t", Description: "d",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := countCalls(f, "os", "Exit"), countCalls(parseProgram(t, harmless), "os", "Exit"); got != want {
		t.Errorf("hostile metadata changed os.Exit call count: got %d, want %d\n%s", got, want, src)
	}
}

// countCalls counts calls to pkg.fn in f.
func countCalls(f *ast.File, pkg, fn string) int {
	n := 0
	ast.Inspect(f, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != fn {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == pkg {
			n++
		}
		return true
	})
	return n
}

func TestRenderProgramOptionalSections(t *testing.T) {
	base := genProgram{Package: "example.com/m/pkg", Type: "Thing", Indent: true}

	plain, err := renderProgram(base)
	if err != nil {
		t.Fatal(err)
	}
	parseProgram(t, plain)
	for _, absent := range []string{"AddGoComments", "Chdir", "schema.ID", "schema.Title", "schema.Description"} {
		if strings.Contains(string(plain), absent) {
			t.Errorf("default program should not contain %q:\n%s", absent, plain)
		}
	}
	if !strings.Contains(string(plain), "json.MarshalIndent") {
		t.Errorf("Indent=true should use MarshalIndent:\n%s", plain)
	}

	compact := base
	compact.Indent = false
	src, err := renderProgram(compact)
	if err != nil {
		t.Fatal(err)
	}
	parseProgram(t, src)
	if strings.Contains(string(src), "MarshalIndent") || !strings.Contains(string(src), "json.Marshal(schema)") {
		t.Errorf("Indent=false should use Marshal:\n%s", src)
	}

	withComments := base
	withComments.Comments = true
	withComments.ModDir = "/work/m"
	withComments.ModName = "example.com/m"
	withComments.RelPkgDir = "./pkg"
	src, err = renderProgram(withComments)
	if err != nil {
		t.Fatal(err)
	}
	parseProgram(t, src)
	for _, want := range []string{`os.Chdir("/work/m")`, `r.AddGoComments("example.com/m", "./pkg")`} {
		if !strings.Contains(string(src), want) {
			t.Errorf("comments program missing %q:\n%s", want, src)
		}
	}
	// Comments must be loaded before reflecting, or they are ignored.
	if strings.Index(string(src), "AddGoComments") > strings.Index(string(src), "r.Reflect") {
		t.Errorf("AddGoComments must precede Reflect:\n%s", src)
	}
}

func TestRelPackageDir(t *testing.T) {
	tests := []struct {
		pkg, mod, want string
		wantErr        bool
	}{
		{"example.com/m", "example.com/m", ".", false},
		{"example.com/m/pkg", "example.com/m", "./pkg", false},
		{"example.com/m/a/b", "example.com/m", "./a/b", false},
		{"example.com/other/pkg", "example.com/m", "", true},
		{"example.com/mm/pkg", "example.com/m", "", true}, // shared prefix is not containment
	}
	for _, tt := range tests {
		got, err := relPackageDir(tt.pkg, tt.mod)
		if tt.wantErr {
			if err == nil {
				t.Errorf("relPackageDir(%q, %q) = %q, want error", tt.pkg, tt.mod, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("relPackageDir(%q, %q) = %q, %v; want %q", tt.pkg, tt.mod, got, err, tt.want)
		}
	}
}
