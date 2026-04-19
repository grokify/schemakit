package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var (
	docOutput  string
	docTitle   string
	docVersion string
	docPrepend string
)

func init() {
	rootCmd.AddCommand(docCmd)

	docCmd.Flags().StringVarP(&docOutput, "output", "o", "", "Output file (default: stdout)")
	docCmd.Flags().StringVarP(&docTitle, "title", "t", "", "Specification title (default: derived from type name)")
	docCmd.Flags().StringVarP(&docVersion, "version", "v", "", "Specification version (e.g., v0.4.0)")
	docCmd.Flags().StringVar(&docPrepend, "prepend", "", "Prepend content from this file (for custom headers/examples)")
}

var docCmd = &cobra.Command{
	Use:   "doc <package> <type>",
	Short: "Generate Markdown documentation from Go struct type",
	Long: `Generate Markdown specification documentation from a Go struct type.

This command creates a temporary Go program that uses reflection to extract
type information, field names, comments, and JSON tags to generate documentation.

Examples:
  # Generate spec for ThreatModel type
  schemakit doc github.com/grokify/threat-model-spec/ir ThreatModel

  # Generate with title and version
  schemakit doc -t "Threat Model Specification" -v v0.4.0 \
    github.com/grokify/threat-model-spec/ir ThreatModel -o spec.md

  # Generate with custom header prepended
  schemakit doc --prepend header.md \
    github.com/grokify/threat-model-spec/ir ThreatModel -o spec.md

Output includes:
  - Type overview with package doc comment
  - Required and optional fields in tables
  - Enum types with values and descriptions
  - Nested type documentation
  - JSON field names and tags`,
	Args: cobra.ExactArgs(2),
	RunE: runDoc,
}

const docTemplate = `//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	target "{{.Package}}"
)

type FieldInfo struct {
	Name        string   ` + "`json:\"name\"`" + `
	JSONName    string   ` + "`json:\"jsonName\"`" + `
	Type        string   ` + "`json:\"type\"`" + `
	Required    bool     ` + "`json:\"required\"`" + `
	Description string   ` + "`json:\"description\"`" + `
	EnumValues  []string ` + "`json:\"enumValues,omitempty\"`" + `
}

type TypeInfo struct {
	Name        string      ` + "`json:\"name\"`" + `
	Package     string      ` + "`json:\"package\"`" + `
	Description string      ` + "`json:\"description\"`" + `
	Fields      []FieldInfo ` + "`json:\"fields\"`" + `
	IsEnum      bool        ` + "`json:\"isEnum\"`" + `
	EnumValues  []string    ` + "`json:\"enumValues,omitempty\"`" + `
}

type EnumValue struct {
	Name        string ` + "`json:\"name\"`" + `
	Value       string ` + "`json:\"value\"`" + `
	Description string ` + "`json:\"description\"`" + `
}

type EnumInfo struct {
	Name        string      ` + "`json:\"name\"`" + `
	Description string      ` + "`json:\"description\"`" + `
	Values      []EnumValue ` + "`json:\"values\"`" + `
}

type DocOutput struct {
	RootType   string     ` + "`json:\"rootType\"`" + `
	PackageDoc string     ` + "`json:\"packageDoc\"`" + `
	Types      []TypeInfo ` + "`json:\"types\"`" + `
	Enums      []EnumInfo ` + "`json:\"enums\"`" + `
}

var processedTypes = make(map[string]bool)
var typeInfos []TypeInfo
var enumInfos []EnumInfo
var fieldComments = make(map[string]map[string]string)
var enumTypes = make(map[string]bool)           // tracks which type names are enums
var enumValues = make(map[string][]EnumValue)   // type name -> values
var typeDescriptions = make(map[string]string)  // type name -> description

func main() {
	// Parse the package for doc comments, enums, and field comments
	parsePackage()

	// Start with the root type
	rootType := reflect.TypeOf(target.{{.Type}}{})
	processType(rootType)

	// Get package doc
	pkgDoc := getPackageDoc()

	// Build enum infos from discovered enums
	for typeName := range enumTypes {
		enumInfos = append(enumInfos, EnumInfo{
			Name:        typeName,
			Description: typeDescriptions[typeName],
			Values:      enumValues[typeName],
		})
	}

	output := DocOutput{
		RootType:   "{{.Type}}",
		PackageDoc: pkgDoc,
		Types:      typeInfos,
		Enums:      enumInfos,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func parsePackage() {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	pkgDir := filepath.Join(gopath, "src", "{{.Package}}")

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, nil, parser.ParseComments)
	if err != nil {
		return
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			parseFile(file)
		}
	}
}

func parseFile(file *ast.File) {
	// Track type declarations and their comments
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		switch genDecl.Tok {
		case token.TYPE:
			parseTypeDecl(genDecl)
		case token.CONST:
			parseConstDecl(genDecl)
		}
	}
}

func parseTypeDecl(genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		typeName := typeSpec.Name.Name

		// Get type description from doc comment
		if genDecl.Doc != nil {
			typeDescriptions[typeName] = strings.TrimSpace(genDecl.Doc.Text())
		}

		// Check if this is a string type alias (potential enum)
		if ident, ok := typeSpec.Type.(*ast.Ident); ok {
			if ident.Name == "string" {
				enumTypes[typeName] = true
				enumValues[typeName] = []EnumValue{} // initialize
			}
		}

		// Parse struct field comments
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		if fieldComments[typeName] == nil {
			fieldComments[typeName] = make(map[string]string)
		}
		for _, field := range structType.Fields.List {
			if field.Doc != nil && len(field.Names) > 0 {
				fieldComments[typeName][field.Names[0].Name] = strings.TrimSpace(field.Doc.Text())
			}
		}
	}
}

func parseConstDecl(genDecl *ast.GenDecl) {
	// Track the current type for iota-style const blocks
	var currentType string

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// Determine the type of this const
		var constType string
		if valueSpec.Type != nil {
			if ident, ok := valueSpec.Type.(*ast.Ident); ok {
				constType = ident.Name
				currentType = constType
			}
		} else {
			constType = currentType
		}

		// Skip if not an enum type we're tracking
		if !enumTypes[constType] {
			continue
		}

		// Extract the const value
		for i, name := range valueSpec.Names {
			if !name.IsExported() {
				continue
			}

			var value string
			if i < len(valueSpec.Values) {
				if lit, ok := valueSpec.Values[i].(*ast.BasicLit); ok {
					value = strings.Trim(lit.Value, "\"")
				}
			}

			// Get description from comment
			desc := ""
			if valueSpec.Doc != nil {
				desc = strings.TrimSpace(valueSpec.Doc.Text())
			} else if valueSpec.Comment != nil {
				desc = strings.TrimSpace(valueSpec.Comment.Text())
			}

			enumValues[constType] = append(enumValues[constType], EnumValue{
				Name:        name.Name,
				Value:       value,
				Description: desc,
			})
		}
	}
}

func getPackageDoc() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	pkgDir := filepath.Join(gopath, "src", "{{.Package}}")

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, nil, parser.ParseComments)
	if err != nil {
		return ""
	}

	for _, pkg := range pkgs {
		d := doc.New(pkg, "{{.Package}}", 0)
		return strings.TrimSpace(d.Doc)
	}
	return ""
}

func processType(t reflect.Type) {
	// Handle pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Skip non-struct types and already processed
	if t.Kind() != reflect.Struct {
		return
	}

	typeName := t.Name()
	if typeName == "" || processedTypes[typeName] {
		return
	}
	processedTypes[typeName] = true

	var fields []FieldInfo
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}

		required := !strings.Contains(jsonTag, "omitempty")

		// Get field type as string
		fieldType := formatType(field.Type)

		// Get description from parsed comments
		desc := ""
		if comments, ok := fieldComments[typeName]; ok {
			desc = comments[field.Name]
		}

		fi := FieldInfo{
			Name:        field.Name,
			JSONName:    jsonName,
			Type:        fieldType,
			Required:    required,
			Description: desc,
		}

		fields = append(fields, fi)

		// Process nested struct types
		processNestedType(field.Type)
	}

	typeInfos = append(typeInfos, TypeInfo{
		Name:        typeName,
		Package:     t.PkgPath(),
		Description: typeDescriptions[typeName],
		Fields:      fields,
	})
}

func processNestedType(t reflect.Type) {
	switch t.Kind() {
	case reflect.Ptr:
		processNestedType(t.Elem())
	case reflect.Slice, reflect.Array:
		processNestedType(t.Elem())
	case reflect.Map:
		processNestedType(t.Elem())
	case reflect.Struct:
		if t.PkgPath() != "" && !strings.HasPrefix(t.PkgPath(), "time") {
			processType(t)
		}
	}
}

func formatType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return formatType(t.Elem())
	case reflect.Slice:
		return "[]" + formatType(t.Elem())
	case reflect.Map:
		return "map[" + formatType(t.Key()) + "]" + formatType(t.Elem())
	case reflect.Struct:
		if t.Name() != "" {
			return t.Name()
		}
		return "object"
	default:
		if t.Name() != "" {
			return t.Name()
		}
		return t.Kind().String()
	}
}
`

func runDoc(cmd *cobra.Command, args []string) error {
	pkgPath := args[0]
	typeName := args[1]

	// Validate type name starts with uppercase (exported)
	if len(typeName) == 0 || typeName[0] < 'A' || typeName[0] > 'Z' {
		return fmt.Errorf("type name must be exported (start with uppercase): %s", typeName)
	}

	// Find the module root and module name
	modRoot, modName := findModule(pkgPath)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "schemakit-doc-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate the temporary program
	tmpl, err := template.New("doc").Parse(docTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]any{
		"Package": pkgPath,
		"Type":    typeName,
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write the temporary program
	docFile := filepath.Join(tmpDir, "doc.go")
	if err := os.WriteFile(docFile, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Helper function to run go commands
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
	if err := goCmd("mod", "init", "schemakit-doc"); err != nil {
		return err
	}

	// Fetch the target module
	if modRoot != "" {
		if err := goCmd("mod", "edit", "-replace", modName+"="+modRoot); err != nil {
			return err
		}
		if err := goCmd("get", pkgPath); err != nil {
			return err
		}
	} else {
		if err := goCmd("get", modName+"@latest"); err != nil {
			return err
		}
	}

	// Run the doc generator
	docGenCmd := exec.Command("go", "run", "doc.go")
	docGenCmd.Dir = tmpDir
	docGenCmd.Env = append(os.Environ(), "GO111MODULE=on")
	var stdout, stderr bytes.Buffer
	docGenCmd.Stdout = &stdout
	docGenCmd.Stderr = &stderr

	if err := docGenCmd.Run(); err != nil {
		return fmt.Errorf("failed to generate docs: %w\n%s", err, stderr.String())
	}

	// Parse the JSON output and generate Markdown
	markdown, err := jsonToMarkdown(stdout.Bytes(), typeName, docTitle, docVersion)
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Prepend custom header if specified
	if docPrepend != "" {
		header, err := os.ReadFile(docPrepend)
		if err != nil {
			return fmt.Errorf("failed to read prepend file: %w", err)
		}
		markdown = string(header) + "\n" + markdown
	}

	// Output result
	if docOutput != "" {
		if err := os.WriteFile(docOutput, []byte(markdown), 0600); err != nil { //nolint:gosec // G703: Path from CLI flag
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Generated %s\n", docOutput)
	} else {
		fmt.Print(markdown)
	}

	return nil
}

// jsonToMarkdown converts the JSON type info to Markdown documentation
func jsonToMarkdown(jsonData []byte, rootType, title, version string) (string, error) {
	var data struct {
		RootType   string `json:"rootType"`
		PackageDoc string `json:"packageDoc"`
		Types      []struct {
			Name        string `json:"name"`
			Package     string `json:"package"`
			Description string `json:"description"`
			Fields      []struct {
				Name        string   `json:"name"`
				JSONName    string   `json:"jsonName"`
				Type        string   `json:"type"`
				Required    bool     `json:"required"`
				Description string   `json:"description"`
				EnumValues  []string `json:"enumValues,omitempty"`
			} `json:"fields"`
		} `json:"types"`
		Enums []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Values      []struct {
				Name        string `json:"name"`
				Value       string `json:"value"`
				Description string `json:"description"`
			} `json:"values"`
		} `json:"enums"`
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", err
	}

	var sb strings.Builder

	// Title
	if title == "" {
		title = rootType + " Specification"
	}
	sb.WriteString("# " + title)
	if version != "" {
		sb.WriteString(" " + version)
	}
	sb.WriteString("\n\n")

	// Package description
	if data.PackageDoc != "" {
		sb.WriteString(data.PackageDoc + "\n\n")
	}

	// Table of contents - Types
	sb.WriteString("## Types\n\n")
	for _, t := range data.Types {
		fmt.Fprintf(&sb, "- [%s](#%s)\n", t.Name, strings.ToLower(t.Name))
	}
	sb.WriteString("\n")

	// Table of contents - Enums (if any)
	if len(data.Enums) > 0 {
		sb.WriteString("## Enums\n\n")
		for _, e := range data.Enums {
			fmt.Fprintf(&sb, "- [%s](#%s)\n", e.Name, strings.ToLower(e.Name))
		}
		sb.WriteString("\n")
	}

	// Type documentation
	sb.WriteString("---\n\n")
	sb.WriteString("# Type Reference\n\n")

	for _, t := range data.Types {
		fmt.Fprintf(&sb, "## %s\n\n", t.Name)

		if t.Description != "" {
			sb.WriteString(t.Description + "\n\n")
		}

		if len(t.Fields) == 0 {
			continue
		}

		// Count required and optional fields
		var hasRequired, hasOptional bool
		for _, f := range t.Fields {
			if f.Required {
				hasRequired = true
			} else {
				hasOptional = true
			}
		}

		// Required fields table
		if hasRequired {
			sb.WriteString("### Required Fields\n\n")
			sb.WriteString("| Field | Type | Description |\n")
			sb.WriteString("|-------|------|-------------|\n")
			for _, f := range t.Fields {
				if !f.Required {
					continue
				}
				desc := f.Description
				if desc == "" {
					desc = "-"
				}
				// Escape pipes in description
				desc = strings.ReplaceAll(desc, "|", "\\|")
				// Truncate long descriptions
				if len(desc) > 100 {
					desc = desc[:97] + "..."
				}
				fmt.Fprintf(&sb, "| `%s` | %s | %s |\n", f.JSONName, f.Type, desc)
			}
			sb.WriteString("\n")
		}

		// Optional fields table
		if hasOptional {
			sb.WriteString("### Optional Fields\n\n")
			sb.WriteString("| Field | Type | Description |\n")
			sb.WriteString("|-------|------|-------------|\n")
			for _, f := range t.Fields {
				if f.Required {
					continue
				}
				desc := f.Description
				if desc == "" {
					desc = "-"
				}
				desc = strings.ReplaceAll(desc, "|", "\\|")
				if len(desc) > 100 {
					desc = desc[:97] + "..."
				}
				fmt.Fprintf(&sb, "| `%s` | %s | %s |\n", f.JSONName, f.Type, desc)
			}
			sb.WriteString("\n")
		}
	}

	// Enum documentation
	if len(data.Enums) > 0 {
		sb.WriteString("---\n\n")
		sb.WriteString("# Enum Reference\n\n")

		for _, e := range data.Enums {
			fmt.Fprintf(&sb, "## %s\n\n", e.Name)

			if e.Description != "" {
				sb.WriteString(e.Description + "\n\n")
			}

			if len(e.Values) == 0 {
				sb.WriteString("*No values defined*\n\n")
				continue
			}

			sb.WriteString("| Value | Description |\n")
			sb.WriteString("|-------|-------------|\n")
			for _, v := range e.Values {
				desc := v.Description
				if desc == "" {
					desc = "-"
				}
				desc = strings.ReplaceAll(desc, "|", "\\|")
				// Use the actual string value if available, otherwise the const name
				displayValue := v.Value
				if displayValue == "" {
					displayValue = v.Name
				}
				fmt.Fprintf(&sb, "| `%s` | %s |\n", displayValue, desc)
			}
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
