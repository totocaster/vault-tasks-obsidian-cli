package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/totocaster/vault-tasks-obsidian-cli/internal/vaulttasks"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "--help", "-h":
			printRootHelp()
			return nil
		case "--version", "-version":
			fmt.Println(versionString())
			return nil
		}
	}

	command := "show"
	commandArgs := args

	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command = args[0]
		commandArgs = args[1:]
	}

	switch command {
	case "show":
		return runShow(commandArgs)
	case "sections":
		return runSections(commandArgs)
	case "settings":
		return runSettings(commandArgs)
	case "version":
		fmt.Println(versionString())
		return nil
	case "help", "-h", "--help":
		printRootHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func runShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	vaultFlag := fs.String("vault", "", "Path to the Obsidian vault")
	filterFlag := fs.String("filter", "", "Task filter: pending, completed, all")
	sectionFlag := fs.String("section", "", "Section filter heading or none")
	connectionsFlag := fs.Bool("connections", false, "Show related notes")
	noConnectionsFlag := fs.Bool("no-connections", false, "Hide related notes")
	formatFlag := fs.String("format", "view", "Output format: view, summary, json")
	widthFlag := fs.String("width", "", "Width mode: readable, full")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *connectionsFlag && *noConnectionsFlag {
		return errors.New("use either --connections or --no-connections, not both")
	}

	vaultPath, err := resolveVaultPath(*vaultFlag)
	if err != nil {
		return err
	}

	env, err := vaulttasks.LoadEnvironment(vaultPath)
	if err != nil {
		return err
	}

	options, err := buildShowOptions(env, *filterFlag, *sectionFlag, *formatFlag, *widthFlag)
	if err != nil {
		return err
	}

	switch {
	case *connectionsFlag:
		options.ShowConnections = true
	case *noConnectionsFlag:
		options.ShowConnections = false
	}

	snapshot, err := vaulttasks.BuildSnapshot(env, options)
	if err != nil {
		return err
	}

	output, err := vaulttasks.RenderShow(snapshot, options)
	if err != nil {
		return err
	}

	fmt.Print(output)
	return nil
}

func runSections(args []string) error {
	fs := flag.NewFlagSet("sections", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	vaultFlag := fs.String("vault", "", "Path to the Obsidian vault")
	filterFlag := fs.String("filter", "", "Task filter: pending, completed, all")

	if err := fs.Parse(args); err != nil {
		return err
	}

	vaultPath, err := resolveVaultPath(*vaultFlag)
	if err != nil {
		return err
	}

	env, err := vaulttasks.LoadEnvironment(vaultPath)
	if err != nil {
		return err
	}

	filter, err := vaulttasks.ResolveFilter(env.Settings, *filterFlag)
	if err != nil {
		return err
	}

	width, err := vaulttasks.ResolveWidth(env.App.ReadableLineLength, "")
	if err != nil {
		return err
	}

	snapshot, err := vaulttasks.BuildSnapshot(env, vaulttasks.ShowOptions{
		Filter:          filter,
		SectionFilter:   nil,
		ShowConnections: false,
		Format:          vaulttasks.FormatView,
		Width:           width,
	})
	if err != nil {
		return err
	}

	fmt.Print(vaulttasks.RenderSections(snapshot))
	return nil
}

func runSettings(args []string) error {
	fs := flag.NewFlagSet("settings", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	vaultFlag := fs.String("vault", "", "Path to the Obsidian vault")

	if err := fs.Parse(args); err != nil {
		return err
	}

	vaultPath, err := resolveVaultPath(*vaultFlag)
	if err != nil {
		return err
	}

	env, err := vaulttasks.LoadEnvironment(vaultPath)
	if err != nil {
		return err
	}

	fmt.Print(vaulttasks.RenderSettings(env))
	return nil
}

func buildShowOptions(
	env *vaulttasks.Environment,
	filterValue string,
	sectionValue string,
	formatValue string,
	widthValue string,
) (vaulttasks.ShowOptions, error) {
	filter, err := vaulttasks.ResolveFilter(env.Settings, filterValue)
	if err != nil {
		return vaulttasks.ShowOptions{}, err
	}

	sectionFilter, err := vaulttasks.ResolveSectionFilter(env.Settings, sectionValue)
	if err != nil {
		return vaulttasks.ShowOptions{}, err
	}

	format, err := vaulttasks.ResolveFormat(formatValue)
	if err != nil {
		return vaulttasks.ShowOptions{}, err
	}

	width, err := vaulttasks.ResolveWidth(env.App.ReadableLineLength, widthValue)
	if err != nil {
		return vaulttasks.ShowOptions{}, err
	}

	return vaulttasks.ShowOptions{
		Filter:          filter,
		SectionFilter:   sectionFilter,
		ShowConnections: env.Settings.ShowConnectionsByDefault,
		Format:          format,
		Width:           width,
	}, nil
}

func resolveVaultPath(value string) (string, error) {
	if strings.TrimSpace(value) != "" {
		return expandPath(value)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	resolved, err := vaulttasks.FindVaultRoot(cwd)
	if err != nil {
		return "", errors.New("could not find an Obsidian vault from the current directory; pass --vault PATH")
	}

	return resolved, nil
}

func expandPath(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	if value == "~" || strings.HasPrefix(value, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if value == "~" {
			value = home
		} else {
			value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
		}
	}

	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}

	return absolute, nil
}

func printRootHelp() {
	fmt.Println(`vault-tasks

Usage:
  vault-tasks [show] [--filter pending|completed|all] [--section <heading>|none] [--connections|--no-connections] [--format view|summary|json] [--width readable|full] [--vault PATH]
  vault-tasks sections [--filter pending|completed|all] [--vault PATH]
  vault-tasks settings [--vault PATH]
  vault-tasks version

Flags:
  --version    Print build version

Description:
  Companion CLI for the Vault Tasks Obsidian plugin. It reads the current
  vault's Obsidian app settings and the plugin's saved settings, then renders
  the same grouped task view for terminals, scripts, and AI agents.
  Plugin repo: https://github.com/totocaster/vault-tasks-obsidian`)
}

func versionString() string {
	return fmt.Sprintf("vault-tasks %s (commit %s, built %s)", version, commit, date)
}
