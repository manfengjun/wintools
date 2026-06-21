//go:build !demo

package main

func handleDemoSecondInstance(_ *App, _ []string) bool {
	return false
}
