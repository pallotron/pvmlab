package log

import (
	"strings"

	"github.com/fatih/color"
)

var (
	titleColor = color.New(color.FgCyan, color.Bold)
	stepColor  = color.New(color.FgCyan, color.Bold)
	infoColor  = color.New(color.FgGreen)
	warnColor  = color.New(color.FgYellow)
	errorColor = color.New(color.FgRed, color.Bold)
	cmdColor   = color.New(color.FgWhite)
)

// Title prints a title message.
func Title(format string, a ...any) {
	titleColor.Printf("==> "+format+"\n", a...)
}

// Step prints a major step in the installation process.
func Step(format string, a ...any) {
	stepColor.Printf("\n==> "+format+"\n", a...)
}

// Info prints an informational message.
func Info(format string, a ...any) {
	infoColor.Printf("  -> "+format+"\n", a...)
}

// Warn prints a warning message.
func Warn(format string, a ...any) {
	warnColor.Printf("  -> WARNING: "+format+"\n", a...)
}

// Error prints an error message.
func Error(format string, a ...any) {
	errorColor.Printf("ERROR: "+format+"\n", a...)
}

// Command prints the command being executed.
func Command(name string, args ...string) {
	cmdColor.Printf("  -> Running: %s %s\n", name, strings.Join(args, " "))
}

// Panic prints a panic message.
func Panic(format string, a ...any) {
	errorColor.Printf("\n==> PANIC: "+format+"\n", a...)
}
