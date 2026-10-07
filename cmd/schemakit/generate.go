package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var (
	genOutput      string
	genIndent      bool
	genCheck       bool
	genComments    bool
	genID          string
	genTitle       string
	genDescription string
)

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&genOutput, "output", "o", "", "Output file (default: stdout)")
	generateCmd.Flags().BoolVar(&genIndent, "indent", true, "Indent JSON output")
	generateCmd.Flags().BoolVar(&genCheck, "check", false, "Verify the committed -o file matches freshly generated output; exit non-zero on drift (no write)")
	generateCmd.Flags().BoolVar(&genComments, "comments", false, "Use Go doc comments from the package (and its subdirectories) as property and type descriptions")
	generateCmd.Flags().StringVar(&genID, "id", "", "Set the schema's $id (default: derived from the package path and type)")
	generateCmd.Flags().StringVar(&genTitle, "title", "", "Set the schema's title")
	generateCmd.Flags().StringVar(&genDescription, "description", "", "Set the schema's description")
}

var generateCmd = &cobra.Command{
	Use:   "generate <package> <type>",
	Short: "Generate JSON Schema from Go struct type",
	Long: `Generate a JSON Schema from a Go struct type using reflection.

This command creates a temporary Go program that imports your type and
uses github.com/invopop/jsonschema to generate the schema.

Examples:
  # Generate schema for TaskList type from structured-tasks
  schemakit generate github.com/grokify/structured-tasks/tasks TaskList

  # Generate and save to file
  schemakit generate -o schema.json github.com/myorg/myproject/types Config

  # Generate without indentation
  schemakit generate --indent=false github.com/myorg/myproject/types Config

  # Fail if the committed schema is out of sync with the Go structs (CI drift guard)
  schemakit generate -o schema.json --check github.com/myorg/myproject/types Config

  # Document properties from Go doc comments and set schema metadata
  schemakit generate --comments \
    --id https://example.com/schemas/config.schema.json \
    --title "Config" --description "Application configuration." \
    -o schema.json github.com/myorg/myproject/types Config

Notes:
  - The package must be importable (available locally or via go get)
  - The type must be exported (start with uppercase)
  - Uses struct tags: json, jsonschema, title, description, etc.
  - --comments reads Go doc comments from the target package's directory
    and below, so types from other packages are not described. Type
    comments contribute their first sentence; field comments in full. A
    On fields, a jsonschema description tag takes precedence over a
    comment. Every .go
    file under that directory is parsed, so a syntactically invalid file
    (for example in testdata) makes generation fail.
  - --check requires -o and does not modify the file; it exits 1 on drift`,
	Args: cobra.ExactArgs(2),
	RunE: runGenerate,
}

// genProgram holds the values rendered into the temporary generator program.
// String fields are Go-quoted by the template, so arbitrary CLI input cannot
// alter the program's structure.
type genProgram struct {
	Package string
	Type    string
	Indent  bool

	// Comments enables Go doc comment extraction. ModDir is the directory of
	// the target module, ModName its import path, and RelPkgDir the target
	// package's directory relative to ModDir ("." for the module root).
	Comments  bool
	ModDir    string
	ModName   string
	RelPkgDir string

	ID          string
	Title       string
	Description string
}

const genTemplate = `//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"
	target "{{.Package}}"
)

func main() {
	r := jsonschema.Reflector{
		DoNotReference: false,
		ExpandedStruct: false,
	}
{{- if .Comments}}
	// invopop keys comments by the base import path joined with the directory
	// as given, so the directory must be relative to the module root.
	if err := os.Chdir({{quote .ModDir}}); err != nil {
		fmt.Fprintf(os.Stderr, "error entering module directory: %v\n", err)
		os.Exit(1)
	}
	if err := r.AddGoComments({{quote .ModName}}, {{quote .RelPkgDir}}); err != nil {
		fmt.Fprintf(os.Stderr, "error reading Go comments: %v\n", err)
		os.Exit(1)
	}
{{- end}}
	schema := r.Reflect(&target.{{.Type}}{})
{{- if .ID}}
	schema.ID = jsonschema.ID({{quote .ID}})
{{- end}}
{{- if .Title}}
	schema.Title = {{quote .Title}}
{{- end}}
{{- if .Description}}
	schema.Description = {{quote .Description}}
{{- end}}
	{{if .Indent}}
	data, err := json.MarshalIndent(schema, "", "  ")
	{{else}}
	data, err := json.Marshal(schema)
	{{end}}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
`

var genTmpl = template.Must(template.New("gen").
	Funcs(template.FuncMap{"quote": strconv.Quote}).
	Parse(genTemplate))

// renderProgram renders the temporary generator program's source.
func renderProgram(p genProgram) ([]byte, error) {
	var buf bytes.Buffer
	if err := genTmpl.Execute(&buf, p); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.Bytes(), nil
}

// relPackageDir returns the directory of pkgPath relative to its module root
// ("." for the root package), given the module's import path.
func relPackageDir(pkgPath, modName string) (string, error) {
	if pkgPath == modName {
		return ".", nil
	}
	rest, ok := strings.CutPrefix(pkgPath, modName+"/")
	if !ok {
		return "", fmt.Errorf("package %s is not inside module %s", pkgPath, modName)
	}
	return "./" + rest, nil
}

func runGenerate(cmd *cobra.Command, args []string) error {
	pkgPath := args[0]
	typeName := args[1]

	// Validate type name starts with uppercase (exported)
	if len(typeName) == 0 || typeName[0] < 'A' || typeName[0] > 'Z' {
		return fmt.Errorf("type name must be exported (start with uppercase): %s", typeName)
	}

	// --check compares against a committed file, so an output path is required.
	if genCheck && genOutput == "" {
		return fmt.Errorf("--check requires -o/--output to specify the file to verify against")
	}

	// Find the module root and module name
	modRoot, modName := findModule(pkgPath)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "schemakit-gen-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	prog := genProgram{
		Package:     pkgPath,
		Type:        typeName,
		Indent:      genIndent,
		Comments:    genComments,
		ModName:     modName,
		ID:          genID,
		Title:       genTitle,
		Description: genDescription,
	}
	if genComments {
		rel, err := relPackageDir(pkgPath, modName)
		if err != nil {
			return err
		}
		prog.RelPkgDir = rel
	}
	// The program is written after the module is fetched, because --comments
	// needs the module's directory, which is only known once it resolves.
	genFile := filepath.Join(tmpDir, "gen.go")

	// Helper function to run go commands and check for errors
	goCmd := func(args ...string) error {
		c := exec.Command("go", args...)
		c.Dir = tmpDir
		c.Env = append(os.Environ(), "GO111MODULE=on")
		var stderr bytes.Buffer
		c.Stderr = &stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("go %v failed: %w\n%s", args, err, stderr.String())
		}
		return nil
	}

	// Initialize the go module
	if err := goCmd("mod", "init", "schemakit-gen"); err != nil {
		return err
	}

	// Fetch jsonschema dependency first (before any replace directives)
	if err := goCmd("get", "github.com/invopop/jsonschema@latest"); err != nil {
		return err
	}

	// Fetch the target module
	if modRoot != "" {
		// For local modules: add replace directive, then get the module
		if err := goCmd("mod", "edit", "-replace", modName+"="+modRoot); err != nil {
			return err
		}
		// Use go get with the package path to add the require
		if err := goCmd("get", pkgPath); err != nil {
			return err
		}
	} else {
		// Remote module - fetch from network
		if err := goCmd("get", modName+"@latest"); err != nil {
			return err
		}
	}

	// Note: We skip `go mod tidy` because gen.go has //go:build ignore
	// which causes tidy to remove all requires since it sees no imports.

	if genComments {
		// Resolve the module directory (the local replace target, or the
		// module cache for a remote module) so comments can be read from source.
		dirOut, err := goCmdOutput(tmpDir, "list", "-m", "-f", "{{.Dir}}", modName)
		if err != nil {
			return err
		}
		prog.ModDir = strings.TrimSpace(dirOut)
		if prog.ModDir == "" {
			return fmt.Errorf("could not locate source for module %s; --comments needs the module's source", modName)
		}
	}

	src, err := renderProgram(prog)
	if err != nil {
		return err
	}
	if err := os.WriteFile(genFile, src, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Run the generator
	genCmd := exec.Command("go", "run", "gen.go")
	genCmd.Dir = tmpDir
	genCmd.Env = append(os.Environ(), "GO111MODULE=on")
	var stdout, stderr bytes.Buffer
	genCmd.Stdout = &stdout
	genCmd.Stderr = &stderr

	if err := genCmd.Run(); err != nil {
		return fmt.Errorf("failed to generate schema: %w\n%s", err, stderr.String())
	}

	// Output result
	output := strings.TrimSpace(stdout.String())

	// --check verifies the committed file matches without writing.
	if genCheck {
		return checkSchemaDrift(cmd, genOutput, output)
	}

	if genOutput != "" {
		if err := os.WriteFile(genOutput, []byte(output+"\n"), 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Generated %s\n", genOutput)
	} else {
		fmt.Println(output)
	}

	return nil
}

// goCmdOutput runs a go command in dir and returns its stdout.
func goCmdOutput(dir string, args ...string) (string, error) {
	c := exec.Command("go", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GO111MODULE=on")
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("go %v failed: %w\n%s", args, err, stderr.String())
	}
	return stdout.String(), nil
}

// checkSchemaDrift compares freshly generated schema output against the
// committed file at path. It returns a non-nil error (drift or read failure)
// so the command exits non-zero, or nil when they match. Comparison ignores
// trailing whitespace so it is insensitive to a final newline.
func checkSchemaDrift(cmd *cobra.Command, path, generated string) error {
	existing, err := os.ReadFile(path) //nolint:gosec // G304: path is a CLI-provided output file
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("schema drift: %s does not exist; run without --check to generate it", path)
		}
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	if strings.TrimSpace(string(existing)) == generated {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s is up to date\n", path)
		return nil
	}

	return fmt.Errorf("schema drift: %s is out of sync with the Go structs; regenerate with `schemakit generate -o %s ...`", path, path)
}

// findModule finds the module root directory and module name for a package path.
// Returns (moduleRoot, moduleName).
// If the module is found locally, moduleRoot is the filesystem path.
// If not found locally, moduleRoot is empty and moduleName is set for remote fetch.
func findModule(pkgPath string) (string, string) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}

	// Split package path into parts
	parts := strings.Split(pkgPath, "/")

	// Try progressively shorter paths to find the module root
	for i := len(parts); i > 0; i-- {
		candidate := filepath.Join(parts[:i]...)
		candidatePath := filepath.Join(gopath, "src", candidate)
		goModPath := filepath.Join(candidatePath, "go.mod")

		if _, err := os.Stat(goModPath); err == nil { //nolint:gosec // G703: Path derived from CLI package argument
			// Found go.mod - read the module name
			content, err := os.ReadFile(goModPath) //nolint:gosec // G703: Path derived from CLI package argument
			if err != nil {
				continue
			}
			modName := parseModuleName(string(content))
			if modName != "" {
				return candidatePath, modName
			}
		}
	}

	// Not found locally - assume it's the package path itself (remote module)
	// Try to guess the module name (first 3 parts for github.com/org/repo pattern)
	if len(parts) >= 3 && (parts[0] == "github.com" || parts[0] == "gitlab.com" || parts[0] == "bitbucket.org") {
		return "", strings.Join(parts[:3], "/")
	}

	return "", pkgPath
}

// parseModuleName extracts the module name from go.mod content.
func parseModuleName(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
