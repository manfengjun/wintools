//go:build demo

package main

import (
	"strings"
	"testing"
)

func TestDemoCommandFromArgs(t *testing.T) {
	command, ok := demoCommandFromArgs([]string{"Wintools-demo.exe", "--demo-command=lock"})
	if !ok || command != "lock" {
		t.Fatalf("got command=%q ok=%v", command, ok)
	}
}

func TestDemoCommandRejectsUnknownCommand(t *testing.T) {
	if command, ok := demoCommandFromArgs([]string{"Wintools-demo.exe", "--demo-command=delete-anything"}); ok {
		t.Fatalf("unexpected accepted command %q", command)
	}
}

func TestPythonDemoScriptNavigatesAndDispatchesSafeDemo(t *testing.T) {
	script, err := demoNavigationScript("python-demo")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"#/py-env", "wintools-demo-pyenv"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("script missing %q", expected)
		}
	}
}

func TestOverviewDemoScriptVisitsCoreSurfaces(t *testing.T) {
	script, err := demoNavigationScript("overview-demo")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"#/desktop-lock", "#/py-env", "wintools-demo-settings"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("script missing %q", expected)
		}
	}
}

func TestPasswordScriptRequiresPassword(t *testing.T) {
	if _, err := demoPasswordScript("lock", ""); err == nil {
		t.Fatal("expected empty password to be rejected")
	}
}

func TestPasswordScriptTargetsDemoControls(t *testing.T) {
	script, err := demoPasswordScript("unlock", "classroom-secret")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"data-demo-id", "lock-toggle", "password-input", "password-confirm"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("script missing %q", expected)
		}
	}
	if !strings.Contains(script, `"classroom-secret"`) {
		t.Fatal("password must be JSON encoded in the injected script")
	}
}
