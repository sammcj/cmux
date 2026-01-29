// internal/workspace/template_test.go
package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListBuiltinTemplates(t *testing.T) {
	templates := ListBuiltinTemplates()

	if len(templates) == 0 {
		t.Error("ListBuiltinTemplates should return at least one template")
	}

	// Check for expected templates
	expectedTemplates := []string{"node", "nextjs", "python", "go", "react", "rust"}
	for _, expected := range expectedTemplates {
		found := false
		for _, tmpl := range templates {
			if tmpl == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected template '%s' not found in list", expected)
		}
	}
}

func TestLoadBuiltinTemplate(t *testing.T) {
	templates := ListBuiltinTemplates()

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			tmpl, err := LoadTemplate(name)
			if err != nil {
				t.Fatalf("LoadTemplate(%s) failed: %v", name, err)
			}

			// Verify template has required fields
			if tmpl.Name != name {
				t.Errorf("expected Name = %s, got %s", name, tmpl.Name)
			}

			if len(tmpl.Packages) == 0 {
				t.Errorf("template %s should have at least one package", name)
			}

			if tmpl.ProcessCompose == "" {
				t.Errorf("template %s should have ProcessCompose content", name)
			}

			// Verify ProcessCompose has expected structure
			if !strings.Contains(tmpl.ProcessCompose, "version:") {
				t.Errorf("template %s ProcessCompose should contain 'version:'", name)
			}

			if !strings.Contains(tmpl.ProcessCompose, "processes:") {
				t.Errorf("template %s ProcessCompose should contain 'processes:'", name)
			}

			// Verify code-server is included
			if !strings.Contains(tmpl.ProcessCompose, "code-server") {
				t.Errorf("template %s ProcessCompose should include code-server", name)
			}
		})
	}
}

func TestLoadUnknownTemplate(t *testing.T) {
	// Unknown templates should fall back to node
	tmpl, err := LoadTemplate("unknown-template")
	if err != nil {
		t.Fatalf("LoadTemplate should not fail for unknown template: %v", err)
	}

	// Should return node template as default
	if tmpl.Name != "node" {
		t.Errorf("expected fallback to 'node' template, got %s", tmpl.Name)
	}
}

func TestNodeTemplate(t *testing.T) {
	tmpl, err := LoadTemplate("node")
	if err != nil {
		t.Fatal(err)
	}

	// Check specific packages
	hasNodejs := false
	hasPnpm := false
	for _, pkg := range tmpl.Packages {
		if strings.HasPrefix(pkg, "nodejs@") {
			hasNodejs = true
		}
		if strings.HasPrefix(pkg, "pnpm@") {
			hasPnpm = true
		}
	}

	if !hasNodejs {
		t.Error("node template should include nodejs")
	}
	if !hasPnpm {
		t.Error("node template should include pnpm")
	}

	// Check env
	if tmpl.Env["NODE_ENV"] != "development" {
		t.Error("node template should set NODE_ENV=development")
	}
}

func TestPythonTemplate(t *testing.T) {
	tmpl, err := LoadTemplate("python")
	if err != nil {
		t.Fatal(err)
	}

	// Check specific packages
	hasPython := false
	hasUv := false
	for _, pkg := range tmpl.Packages {
		if strings.HasPrefix(pkg, "python@") {
			hasPython = true
		}
		if strings.HasPrefix(pkg, "uv@") {
			hasUv = true
		}
	}

	if !hasPython {
		t.Error("python template should include python")
	}
	if !hasUv {
		t.Error("python template should include uv")
	}

	// Check env
	if tmpl.Env["PYTHONDONTWRITEBYTECODE"] != "1" {
		t.Error("python template should set PYTHONDONTWRITEBYTECODE=1")
	}

	// Check ProcessCompose contains uvicorn
	if !strings.Contains(tmpl.ProcessCompose, "uvicorn") {
		t.Error("python template ProcessCompose should mention uvicorn")
	}
}

func TestNextjsTemplate(t *testing.T) {
	tmpl, err := LoadTemplate("nextjs")
	if err != nil {
		t.Fatal(err)
	}

	// Check env
	if tmpl.Env["NEXT_TELEMETRY_DISABLED"] != "1" {
		t.Error("nextjs template should disable telemetry")
	}

	// Check ProcessCompose contains next
	if !strings.Contains(tmpl.ProcessCompose, "next") {
		t.Error("nextjs template ProcessCompose should mention next")
	}
}

func TestGoTemplate(t *testing.T) {
	tmpl, err := LoadTemplate("go")
	if err != nil {
		t.Fatal(err)
	}

	// Check specific packages
	hasGo := false
	for _, pkg := range tmpl.Packages {
		if strings.HasPrefix(pkg, "go@") {
			hasGo = true
		}
	}

	if !hasGo {
		t.Error("go template should include go")
	}
}

func TestRustTemplate(t *testing.T) {
	tmpl, err := LoadTemplate("rust")
	if err != nil {
		t.Fatal(err)
	}

	// Check specific packages
	hasRustup := false
	for _, pkg := range tmpl.Packages {
		if strings.HasPrefix(pkg, "rustup@") {
			hasRustup = true
		}
	}

	if !hasRustup {
		t.Error("rust template should include rustup")
	}

	// Check ProcessCompose contains cargo
	if !strings.Contains(tmpl.ProcessCompose, "cargo") {
		t.Error("rust template ProcessCompose should mention cargo")
	}
}

func TestCustomTemplate(t *testing.T) {
	// Create temp directory for custom template
	tmpDir, err := os.MkdirTemp("", "template_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set DBA_HOME to temp directory
	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	// Create custom template directory
	customTmplDir := filepath.Join(tmpDir, "templates", "custom-tmpl")
	if err := os.MkdirAll(customTmplDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create template.yaml
	templateYaml := `
name: custom-tmpl
packages:
  - custom-pkg@1.0
env:
  CUSTOM_VAR: "custom_value"
`
	if err := os.WriteFile(filepath.Join(customTmplDir, "template.yaml"), []byte(templateYaml), 0644); err != nil {
		t.Fatal(err)
	}

	// Create process-compose.yaml
	processCompose := `
version: "0.5"
processes:
  custom:
    command: echo "custom"
`
	if err := os.WriteFile(filepath.Join(customTmplDir, "process-compose.yaml"), []byte(processCompose), 0644); err != nil {
		t.Fatal(err)
	}

	// Load custom template
	tmpl, err := LoadTemplate("custom-tmpl")
	if err != nil {
		t.Fatalf("LoadTemplate(custom-tmpl) failed: %v", err)
	}

	if tmpl.Name != "custom-tmpl" {
		t.Errorf("expected Name = custom-tmpl, got %s", tmpl.Name)
	}

	if len(tmpl.Packages) != 1 || tmpl.Packages[0] != "custom-pkg@1.0" {
		t.Errorf("expected packages = [custom-pkg@1.0], got %v", tmpl.Packages)
	}

	if tmpl.Env["CUSTOM_VAR"] != "custom_value" {
		t.Errorf("expected CUSTOM_VAR = custom_value, got %s", tmpl.Env["CUSTOM_VAR"])
	}

	if !strings.Contains(tmpl.ProcessCompose, "custom") {
		t.Error("ProcessCompose should be loaded from custom template")
	}
}

func TestTemplateProcessComposeHasEnvironmentVariables(t *testing.T) {
	templates := ListBuiltinTemplates()

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			tmpl, err := LoadTemplate(name)
			if err != nil {
				t.Fatal(err)
			}

			// All templates should use ${CODE_PORT} and ${PORT} variables
			if !strings.Contains(tmpl.ProcessCompose, "${CODE_PORT}") {
				t.Errorf("template %s ProcessCompose should use ${CODE_PORT}", name)
			}

			if !strings.Contains(tmpl.ProcessCompose, "${PORT}") {
				t.Errorf("template %s ProcessCompose should use ${PORT}", name)
			}

			if !strings.Contains(tmpl.ProcessCompose, "${DBA_WORKSPACE_PATH}") {
				t.Errorf("template %s ProcessCompose should use ${DBA_WORKSPACE_PATH}", name)
			}
		})
	}
}
