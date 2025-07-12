package display

import (
	"os"

	"golang.org/x/term"
)

// GetTerminalWidth returns the current terminal width in characters
// Returns 80 as fallback if detection fails
func GetTerminalWidth() int {
	fd := int(os.Stdout.Fd())
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return 80 // Fallback to standard width
	}
	return width
}

// GetTerminalSize returns both width and height of the terminal
// Returns (80, 24) as fallback if detection fails
func GetTerminalSize() (width, height int) {
	fd := int(os.Stdout.Fd())
	w, h, err := term.GetSize(fd)
	if err != nil || w <= 0 || h <= 0 {
		return 80, 24 // Fallback to standard size
	}
	return w, h
}