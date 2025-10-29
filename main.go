package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

type AppEntry struct {
	Name string
	Path string
}

func main() {
	args := ParseArgs()

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: using default config -", err)
		cfg = defaultConfig()
	}

	var items []string
	var appEntries []AppEntry

	switch args.Mode.Value {
	case "dmenu":
		// Ensure piped input
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Fprintln(os.Stderr, "Error: expected piped input, e.g., `ls | greg -m dmenu`.")
			os.Exit(1)
		}

		// Read piped items
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			items = append(items, scanner.Text())
		}

	case "menu":
		cfg.MaxItems = getMaxItems(cfg)

		mnu, err := loadMenu()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		runMenu(mnu.Menu, cfg, args)
		os.Exit(0)

	// Use apps mode if no mode is specified
	case "apps":
		fallthrough
	case "":
		args.Mode.Value = "apps"
		appEntries, err = readDesktopFiles("/home/chris/.local/share/applications")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading .desktop files:", err)
			os.Exit(1)
		}

		for _, app := range appEntries {
			items = append(items, app.Name)

			// Print debug info
			if cfg.Log {
				fmt.Printf("[DEBUG] Loaded app: %s (%s)\n", app.Name, app.Path)
			}
		}

	default:
		fmt.Fprintln(os.Stderr, "Error: unknown mode. Supported modes: dmenu, apps")
		os.Exit(1)
	}

	cfg.MaxItems = getMaxItems(cfg)

	if cfg.Log {
		fmt.Printf("[DEBUG] Total apps loaded: %d\n", len(appEntries))
	}

	mode := initialModelWithItems(cfg, args, items)
	if _, err := RunTUIWithItems(cfg, mode, items, appEntries); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// readDesktopFiles returns the "Name=" entries from all .desktop files in the folder
func readDesktopFiles(dir string) ([]AppEntry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var apps []AppEntry
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".desktop") {
			continue
		}

		path := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var name string
		for line := range strings.SplitSeq(string(data), "\n") {
			if after, ok := strings.CutPrefix(line, "Name="); ok {
				name = after
				name = strings.TrimSpace(name)
				break
			}
		}
		if name != "" {
			apps = append(apps, AppEntry{Name: name, Path: path})
		}
	}

	return apps, nil
}

// getMaxItems calculates the number of visible items for the TUI.
// If cfg.MaxItems >= 0, it returns cfg.MaxItems.
// If cfg.MaxItems == -1, it auto-detects terminal height.
func getMaxItems(cfg *Config) int {
	if cfg.MaxItems > 0 {
		return cfg.MaxItems
	}

	height, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil || height < 5 {
		// Fallback if detection fails
		if cfg.Log {
			fmt.Printf("[DEBUG] Failed to get terminal size, using default max items 10: %v\n", err)
		}
		return cfg.DefaultMaxItems
	}

	// Reserve lines for header, prompt, margins, etc.
	reservedLines := 4
	maxItems := max(height-reservedLines, 1)
	return maxItems
}
