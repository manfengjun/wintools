//go:build demo

package main

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var allowedDemoCommands = map[string]bool{
	"lock":          true,
	"unlock":        true,
	"show":          true,
	"minimize":      true,
	"python-demo":   true,
	"overview-demo": true,
}

func demoNavigationScript(command string) (string, error) {
	switch command {
	case "python-demo":
		return `location.hash = '#/py-env'; setTimeout(() => window.dispatchEvent(new CustomEvent('wintools-demo-pyenv')), 1200);`, nil
	case "overview-demo":
		return `location.hash = '#/desktop-lock'; setTimeout(() => { location.hash = '#/py-env'; }, 4500); setTimeout(() => window.dispatchEvent(new CustomEvent('wintools-demo-settings')), 9500);`, nil
	default:
		return "", errors.New("unsupported demo navigation command")
	}
}

func demoCommandFromArgs(args []string) (string, bool) {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--demo-command=") {
			continue
		}
		command := strings.TrimPrefix(arg, "--demo-command=")
		return command, allowedDemoCommands[command]
	}
	return "", false
}

func demoPasswordScript(command, password string) (string, error) {
	if command != "lock" && command != "unlock" {
		return "", errors.New("password script only supports lock and unlock")
	}
	if password == "" {
		return "", errors.New("WINTOOLS_DEMO_PASSWORD is required")
	}
	encodedPassword, _ := json.Marshal(password)
	return `(function () {
  const toggle = document.querySelector('[data-demo-id="lock-toggle"]');
  if (!toggle) throw new Error('demo lock toggle not found');
  toggle.click();
  setTimeout(() => {
    const input = document.querySelector('[data-demo-id="password-input"]');
    const confirm = document.querySelector('[data-demo-id="password-confirm"]');
    if (!input || !confirm) throw new Error('demo password dialog not found');
    const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
    setter.call(input, ` + string(encodedPassword) + `);
    input.dispatchEvent(new Event('input', { bubbles: true }));
    confirm.click();
  }, 350);
})();`, nil
}

func handleDemoSecondInstance(app *App, args []string) bool {
	command, ok := demoCommandFromArgs(args)
	if !ok || app.ctx == nil {
		return false
	}

	switch command {
	case "show":
		app.showMainWindow()
	case "minimize":
		wailsRuntime.WindowMinimise(app.ctx)
	case "lock", "unlock":
		script, err := demoPasswordScript(command, os.Getenv("WINTOOLS_DEMO_PASSWORD"))
		if err != nil {
			return true
		}
		app.showMainWindow()
		wailsRuntime.WindowExecJS(app.ctx, script)
	case "python-demo", "overview-demo":
		script, err := demoNavigationScript(command)
		if err != nil {
			return true
		}
		app.showMainWindow()
		wailsRuntime.WindowExecJS(app.ctx, script)
	}
	return true
}
