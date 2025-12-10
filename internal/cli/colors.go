package cli

import (
	"fmt"
	"os"
	"runtime"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Underline = "\033[4m"

	// Colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright colors
	BrightBlack   = "\033[90m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// 256 color mode - Orange
	Orange = "\033[38;5;208m"

	// Background colors
	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgBlue   = "\033[44m"
)

// Symbols for status indicators
const (
	SymbolCheck   = "✓"
	SymbolCross   = "✗"
	SymbolWarning = "⚠"
	SymbolInfo    = "ℹ"
	SymbolArrow   = "→"
	SymbolDot     = "•"
	SymbolStar    = "★"
)

// colorsEnabled indicates whether color output is enabled
var colorsEnabled = true

func init() {
	// Disable colors if NO_COLOR env is set or not a terminal
	if os.Getenv("NO_COLOR") != "" {
		colorsEnabled = false
		return
	}

	// Check if stdout is a terminal
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		colorsEnabled = false
		return
	}

	// Windows cmd.exe doesn't support ANSI by default (but Windows Terminal does)
	if runtime.GOOS == "windows" {
		// Could add Windows console API calls here, but for now just check TERM
		if os.Getenv("TERM") == "" && os.Getenv("WT_SESSION") == "" {
			colorsEnabled = false
		}
	}
}

// DisableColors disables color output
func DisableColors() {
	colorsEnabled = false
}

// EnableColors enables color output
func EnableColors() {
	colorsEnabled = true
}

// colorize wraps text with color codes if colors are enabled
func colorize(color, text string) string {
	if !colorsEnabled {
		return text
	}
	return color + text + Reset
}

// Style functions

// Success returns green text with checkmark
func Success(text string) string {
	if !colorsEnabled {
		return "[OK] " + text
	}
	return colorize(Green, SymbolCheck+" "+text)
}

// Error returns red text with cross
func Error(text string) string {
	if !colorsEnabled {
		return "[ERROR] " + text
	}
	return colorize(Red, SymbolCross+" "+text)
}

// Warning returns yellow text with warning symbol
func Warning(text string) string {
	if !colorsEnabled {
		return "[WARN] " + text
	}
	return colorize(Yellow, SymbolWarning+" "+text)
}

// Info returns blue text with info symbol
func Info(text string) string {
	if !colorsEnabled {
		return "[INFO] " + text
	}
	return colorize(Cyan, SymbolInfo+" "+text)
}

// Title returns bold text
func Title(text string) string {
	if !colorsEnabled {
		return "=== " + text + " ==="
	}
	return colorize(Bold+BrightWhite, text)
}

// Subtitle returns dim text
func Subtitle(text string) string {
	return colorize(Dim, text)
}

// Highlight returns bold cyan text
func Highlight(text string) string {
	return colorize(Bold+Cyan, text)
}

// Command returns yellow text for commands
func Command(text string) string {
	return colorize(Yellow, text)
}

// Path returns blue text for file paths
func Path(text string) string {
	return colorize(Blue, text)
}

// URL returns underlined cyan text for URLs
func URL(text string) string {
	if !colorsEnabled {
		return text
	}
	return colorize(Underline+Cyan, text)
}

// Status returns colored status text
func Status(running bool) string {
	if running {
		return colorize(Green, "running")
	}
	return colorize(Red, "stopped")
}

// StatusInstalled returns colored installed status
func StatusInstalled(installed bool) string {
	if installed {
		return colorize(Green, "installed")
	}
	return colorize(Red, "not installed")
}

// Bullet returns a bullet point
func Bullet(text string) string {
	if !colorsEnabled {
		return "  - " + text
	}
	return colorize(Dim, "  "+SymbolDot+" ") + text
}

// Arrow returns an arrow prefix
func Arrow(text string) string {
	if !colorsEnabled {
		return "  -> " + text
	}
	return colorize(Cyan, "  "+SymbolArrow+" ") + text
}

// Header prints a section header
func Header(text string) string {
	if !colorsEnabled {
		return "\n" + text + "\n" + repeatChar('-', len(text))
	}
	return "\n" + colorize(Bold+BrightWhite, text) + "\n" + colorize(Dim, repeatChar('─', len(text)))
}

// Box creates a simple box around text
func Box(text string) string {
	if !colorsEnabled {
		return "+------------------+\n| " + text + " |\n+------------------+"
	}
	line := colorize(Dim, "┌"+repeatChar('─', len(text)+2)+"┐")
	middle := colorize(Dim, "│ ") + text + colorize(Dim, " │")
	bottom := colorize(Dim, "└"+repeatChar('─', len(text)+2)+"┘")
	return line + "\n" + middle + "\n" + bottom
}

// ProgressDot returns a progress dot for spinners
func ProgressDot() string {
	return colorize(Cyan, ".")
}

// LogLevel returns colored log level
func LogLevel(level string) string {
	switch level {
	case "DEBUG":
		return colorize(Dim, level)
	case "INFO":
		return colorize(Blue, level)
	case "NOTICE":
		return colorize(Cyan, level)
	case "WARNING":
		return colorize(Yellow, level)
	case "ERROR":
		return colorize(Red, level)
	case "CRITICAL", "ALERT", "EMERGENCY":
		return colorize(Bold+Red, level)
	default:
		return level
	}
}

// LogFile returns colored filename for log output
func LogFile(filename string) string {
	return colorize(Magenta, filename)
}

// Timestamp returns dimmed timestamp
func Timestamp(ts string) string {
	return colorize(Dim, ts)
}

// repeatChar repeats a character n times
func repeatChar(char rune, n int) string {
	result := make([]rune, n)
	for i := range result {
		result[i] = char
	}
	return string(result)
}

// Sprintf is like fmt.Sprintf but strips colors if disabled
func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}

// Print helpers

// PrintSuccess prints a success message
func PrintSuccess(format string, a ...interface{}) {
	fmt.Println(Success(fmt.Sprintf(format, a...)))
}

// PrintError prints an error message
func PrintError(format string, a ...interface{}) {
	fmt.Println(Error(fmt.Sprintf(format, a...)))
}

// PrintWarning prints a warning message
func PrintWarning(format string, a ...interface{}) {
	fmt.Println(Warning(fmt.Sprintf(format, a...)))
}

// PrintInfo prints an info message
func PrintInfo(format string, a ...interface{}) {
	fmt.Println(Info(fmt.Sprintf(format, a...)))
}

// PrintTitle prints a title
func PrintTitle(format string, a ...interface{}) {
	fmt.Println(Title(fmt.Sprintf(format, a...)))
}

// PrintHeader prints a section header
func PrintHeader(format string, a ...interface{}) {
	fmt.Println(Header(fmt.Sprintf(format, a...)))
}

// PrintLogo prints the MageBox logo with version
func PrintLogo(version string) {
	orange := Orange
	white := BrightWhite
	reset := Reset
	dim := Dim

	if !colorsEnabled {
		orange = ""
		white = ""
		reset = ""
		dim = ""
	}

	logo := `
` + orange + `    ╭───────────────────────────────────────╮` + reset + `
` + orange + `    │` + reset + `                                       ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `███╗   ███╗` + reset + ` ` + white + `█████╗  ██████╗ ███████╗` + reset + `  ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `████╗ ████║` + reset + ` ` + white + `██╔══██╗██╔════╝ ██╔════╝` + reset + `  ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `██╔████╔██║` + reset + ` ` + white + `███████║██║  ███╗█████╗` + reset + `    ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `██║╚██╔╝██║` + reset + ` ` + white + `██╔══██║██║   ██║██╔══╝` + reset + `    ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `██║ ╚═╝ ██║` + reset + ` ` + white + `██║  ██║╚██████╔╝███████╗` + reset + `  ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `╚═╝     ╚═╝` + reset + ` ` + white + `╚═╝  ╚═╝ ╚═════╝ ╚══════╝` + reset + `  ` + orange + `│` + reset + `
` + orange + `    │` + reset + `                                       ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `██████╗  ██████╗ ██╗  ██╗` + reset + `             ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `██╔══██╗██╔═══██╗╚██╗██╔╝` + reset + `             ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `██████╔╝██║   ██║ ╚███╔╝` + reset + `              ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `██╔══██╗██║   ██║ ██╔██╗` + reset + `              ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `██████╔╝╚██████╔╝██╔╝ ██╗` + reset + `             ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + orange + `╚═════╝  ╚═════╝ ╚═╝  ╚═╝` + reset + `             ` + orange + `│` + reset + `
` + orange + `    │` + reset + `                                       ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + dim + `Modern Magento Development` + reset + `            ` + orange + `│` + reset + `
` + orange + `    │` + reset + `   ` + dim + `Version ` + white + version + reset + `                         ` + orange + `│` + reset + `
` + orange + `    │` + reset + `                                       ` + orange + `│` + reset + `
` + orange + `    ╰───────────────────────────────────────╯` + reset + `
`
	fmt.Print(logo)
}

// PrintLogoSmall prints a compact MageBox ASCII logo
func PrintLogoSmall(version string) {
	orange := Orange
	reset := Reset

	if !colorsEnabled {
		orange = ""
		reset = ""
	}

	logo := orange + `                            _
                           | |
 _ __ ___   __ _  __ _  ___| |__   _____  __
| '_ ` + "`" + ` _ \ / _` + "`" + ` |/ _` + "`" + ` |/ _ \ '_ \ / _ \ \/ /
| | | | | | (_| | (_| |  __/ |_) | (_) >  <
|_| |_| |_|\__,_|\__, |\___|_.__/ \___/_/\_\
                  __/ |
                 |___/` + reset + `  ` + version + `
`

	fmt.Print(logo)
}
