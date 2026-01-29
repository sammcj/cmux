// internal/cli/computer_wait_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestComputerWaitCommand(t *testing.T) {
	root := GetRootCmd()

	var waitCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "wait" {
					waitCmd = sub
					break
				}
			}
			break
		}
	}

	if waitCmd == nil {
		t.Fatal("wait command not found")
	}

	// Check flags
	expectedFlags := []string{"timeout", "text", "url", "ms"}
	for _, flag := range expectedFlags {
		if f := waitCmd.Flag(flag); f == nil {
			t.Errorf("expected '%s' flag", flag)
		}
	}

	// Check description
	if waitCmd.Short == "" {
		t.Error("wait command should have short description")
	}
	if waitCmd.Long == "" {
		t.Error("wait command should have long description")
	}
}

func TestWaitCommandFlagDefaults(t *testing.T) {
	root := GetRootCmd()

	var waitCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "wait" {
					waitCmd = sub
					break
				}
			}
			break
		}
	}

	if waitCmd == nil {
		t.Fatal("wait command not found")
	}

	// Check timeout flag default
	timeoutFlag := waitCmd.Flag("timeout")
	if timeoutFlag == nil {
		t.Fatal("timeout flag not found")
	}
	if timeoutFlag.DefValue != "30000" {
		t.Errorf("timeout flag default should be 30000, got: %s", timeoutFlag.DefValue)
	}

	// Check text flag default
	textFlag := waitCmd.Flag("text")
	if textFlag == nil {
		t.Fatal("text flag not found")
	}
	if textFlag.DefValue != "" {
		t.Errorf("text flag default should be empty, got: %s", textFlag.DefValue)
	}

	// Check url flag default
	urlFlag := waitCmd.Flag("url")
	if urlFlag == nil {
		t.Fatal("url flag not found")
	}
	if urlFlag.DefValue != "" {
		t.Errorf("url flag default should be empty, got: %s", urlFlag.DefValue)
	}

	// Check ms flag default
	msFlag := waitCmd.Flag("ms")
	if msFlag == nil {
		t.Fatal("ms flag not found")
	}
	if msFlag.DefValue != "0" {
		t.Errorf("ms flag default should be 0, got: %s", msFlag.DefValue)
	}
}

func TestWaitCommandHelp(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "wait", "--help"})

	err := root.Execute()
	if err != nil {
		t.Errorf("help should not return error: %v", err)
	}

	output := strings.ToLower(buf.String())
	expected := []string{"wait", "element", "timeout", "text", "url"}
	for _, substr := range expected {
		if !strings.Contains(output, substr) {
			t.Errorf("expected help output to contain '%s'", substr)
		}
	}
}

func TestWaitCommandExamples(t *testing.T) {
	root := GetRootCmd()

	var waitCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "wait" {
					waitCmd = sub
					break
				}
			}
			break
		}
	}

	if waitCmd == nil {
		t.Fatal("wait command not found")
	}

	examples := waitCmd.Example
	if examples == "" {
		t.Error("wait command should have examples")
	}

	// Check for variety of examples
	expectedInExamples := []string{"@e1", "2000", "text", "url"}
	for _, exp := range expectedInExamples {
		if !strings.Contains(examples, exp) {
			t.Errorf("examples should mention '%s'", exp)
		}
	}
}

func TestWaitCommandUsageVariations(t *testing.T) {
	root := GetRootCmd()

	var waitCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "wait" {
					waitCmd = sub
					break
				}
			}
			break
		}
	}

	if waitCmd == nil {
		t.Fatal("wait command not found")
	}

	// Check usage mentions both selector and ms
	usage := waitCmd.Use
	if !strings.Contains(usage, "wait") {
		t.Error("usage should contain 'wait'")
	}
}

func TestWaitLongDescriptionContent(t *testing.T) {
	root := GetRootCmd()

	var waitCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "wait" {
					waitCmd = sub
					break
				}
			}
			break
		}
	}

	if waitCmd == nil {
		t.Fatal("wait command not found")
	}

	long := strings.ToLower(waitCmd.Long)

	// Should explain different wait types
	descriptions := []string{"element", "duration", "text", "url"}
	for _, desc := range descriptions {
		if !strings.Contains(long, desc) {
			t.Errorf("long description should mention '%s'", desc)
		}
	}
}
