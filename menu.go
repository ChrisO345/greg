package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"
	tea "github.com/charmbracelet/bubbletea"
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
	// persistent TUI only for menu mode
	if args.Mode.Value == "menu" {
		if err := RunPersistentMenuTUI(cfg, args, menuConfig); err != nil {
			fmt.Fprintln(os.Stderr, "Menu TUI error:", err)
		}
		return
	}

	fmt.Println("Error: runMenu should only be used in menu mode")
}

func expandGenerator(cmdStr string) ([]Menu, error) {
	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("generator failed: %v\n%s", err, out.String())
	}

	var result struct {
		Items []Menu `toml:"items"`
	}

	if _, err := toml.Decode(out.String(), &result); err != nil {
		return nil, fmt.Errorf("decode generator TOML failed: %v\n%s", err, out.String())
	}

	return result.Items, nil
}

func executeCommand(cmdStr string, visible bool) error {
	cmd := exec.Command("/bin/sh", "-c", cmdStr)

	fmt.Println("Executing, state: ", visible)

	if !visible {
		// run detached
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		return cmd.Start()
	}

	// visible foreground run
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func loadMenu() (*MenuConfig, error) {
	config := &MenuConfig{}

	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home dir: %w", err)
		}
		xdgHome = filepath.Join(home, ".config")
	}

	path := filepath.Join(xdgHome, "greg", "menu.toml")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("menu file not found: %s", path)
	}

	if _, err := toml.DecodeFile(path, config); err != nil {
		return nil, fmt.Errorf("error parsing menu file: %w", err)
	}

	return config, nil
}

func RunPersistentMenuTUI(cfg *Config, args *CLIArgs, menu *MenuConfig) error {
	m := initialPersistentMenuModel(cfg, args, menu)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
