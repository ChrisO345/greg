package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"
)

type Menu struct {
	Label     string `toml:"label"`
	Exec      string `toml:"exec,omitempty"`
	Generator string `toml:"generator,omitempty"`
	Prompt    string `toml:"prompt,omitempty"`
	Title     string `toml:"title,omitempty"`
	Visible   bool   `toml:"visible,omitempty"`
	Items     []Menu `toml:"items,omitempty"`
}

type MenuConfig struct {
	Menu   []Menu `toml:"menu"`
	Prompt string `toml:"prompt,omitempty"`
	Title  string `toml:"title,omitempty"`
}

func runMenu(menuConfig *MenuConfig, cfg *Config, args *CLIArgs) {
	fmt.Printf("Running menu...\n")

	menus := menuConfig.Menu
	prompt := menuConfig.Prompt
	title := menuConfig.Title

	previousLabel := "greg"

	for {
		var items []string
		for _, m := range menus {
			items = append(items, m.Label)
		}

		mode := initialModelWithItems(cfg, args, items)
		if prompt != "" {
			mode.prompt = prompt
		}

		if title != "" {
			mode.mainHeader = title
		} else {
			mode.mainHeader = previousLabel
		}

		selected, err := RunTUIWithItems(cfg, mode, items, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		if selected == "" {
			fmt.Println("No selection made, exiting menu.")
			break
		}

		found := false
		for _, m := range menus {
			if m.Label == selected {
				found = true

				switch {
				case len(m.Items) > 0:
					menus = m.Items
					prompt = m.Prompt
					title = m.Title
					previousLabel = m.Label

				case m.Exec != "":
					if err := executeCommand(m.Exec, m.Visible); err != nil {
						fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
					}
					return

				case m.Generator != "":
					fmt.Printf("Running generator: %s\n", m.Generator)
					subitems, err := expandGenerator(m.Generator)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error in generator: %v\n", err)
						os.Exit(1)
					}
					m.Items = subitems
					prompt = m.Prompt
					title = m.Title
					menus = m.Items
					previousLabel = m.Label

				default:
					fmt.Printf("No action for: %s\n", m.Label)
					os.Exit(1)
				}
			}
		}

		if !found {
			fmt.Printf("Selected item not found: %s\n", selected)
			os.Exit(1)
		}
	}
}

func expandGenerator(cmdStr string) ([]Menu, error) {
	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running generator failed: %v\n%s", err, out.String())
	}

	var result struct {
		Items []Menu `toml:"items"`
	}

	if _, err := toml.Decode(out.String(), &result); err != nil {
		return nil, fmt.Errorf("failed to decode generator output: %v\n%s", err, out.String())
	}

	return result.Items, nil
}

func executeCommand(res string, visible bool) error {
	fmt.Printf("%v\n", visible)
	cmd := exec.Command("/bin/sh", "-c", res)
	if !visible {
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		return cmd.Start()
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}
}

func loadMenu() (*MenuConfig, error) {
	config := &MenuConfig{}

	// Determine config file path
	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home dir: %w", err)
		}
		xdgHome = filepath.Join(home, ".config")
	}

	menuPath := filepath.Join(xdgHome, "greg", "menu.toml")

	// If the file doesn't exist, error
	if _, err := os.Stat(menuPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("menu file not found at %s", menuPath)
	}

	// Decode TOML
	if _, err := toml.DecodeFile(menuPath, config); err != nil {
		return nil, fmt.Errorf("error parsing menu: %w", err)
	}

	return config, nil
}
