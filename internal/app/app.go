package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
)

// Config holds CLI options shared by commands.
type Config struct {
	WorkflowDir string
	Lockfile    string
	DefaultHost string
	Format      string
	ConfigPath  string
}

// Run executes the workflow-lock CLI.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("missing command")
	}

	switch args[0] {
	case "lock":
		cfg, err := parseConfig(args[1:])
		if err != nil {
			return err
		}
		return runLock(ctx, cfg, stdout)
	case "verify":
		cfg, err := parseConfig(args[1:])
		if err != nil {
			return err
		}
		return runVerify(ctx, cfg, stdout)
	case "list":
		cfg, err := parseConfig(args[1:])
		if err != nil {
			return err
		}
		return runList(ctx, cfg, stdout)
	case "diff":
		cfg, err := parseConfig(args[1:])
		if err != nil {
			return err
		}
		return runDiff(ctx, cfg, stdout)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func parseConfig(args []string) (Config, error) {
	configPath, err := resolveConfig(args)
	if err != nil {
		return Config{}, err
	}
	fileCfg, err := loadConfigFile(configPath)
	if err != nil {
		return Config{}, err
	}

	fs := flag.NewFlagSet("workflow-lock", flag.ContinueOnError)
	cfg := Config{
		WorkflowDir: ".github/workflows",
		Lockfile:    "workflow-lock.yaml",
		DefaultHost: "github.com",
		Format:      "text",
		ConfigPath:  configPath,
	}
	if fileCfg.WorkflowDir != "" {
		cfg.WorkflowDir = fileCfg.WorkflowDir
	}
	if fileCfg.Lockfile != "" {
		cfg.Lockfile = fileCfg.Lockfile
	}
	if fileCfg.DefaultHost != "" {
		cfg.DefaultHost = fileCfg.DefaultHost
	}
	if fileCfg.Format != "" {
		cfg.Format = fileCfg.Format
	}
	fs.StringVar(&cfg.WorkflowDir, "workflows", cfg.WorkflowDir, "directory containing workflow YAML files")
	fs.StringVar(&cfg.Lockfile, "lockfile", cfg.Lockfile, "path to workflow lock file")
	fs.StringVar(&cfg.DefaultHost, "default-host", cfg.DefaultHost, "default host for plain owner/repo workflow refs")
	fs.StringVar(&cfg.Format, "format", cfg.Format, "output format for list and diff: text or json")
	fs.StringVar(&cfg.ConfigPath, "config", configPath, "path to optional workflow-lock config file")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if cfg.Format != "text" && cfg.Format != "json" {
		return Config{}, fmt.Errorf("unsupported format %q", cfg.Format)
	}
	if fs.NArg() != 0 {
		return Config{}, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	return cfg, nil
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage: workflow-lock <lock|verify|list|diff> [flags]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "flags:")
	_, _ = fmt.Fprintln(w, "  -workflows string")
	_, _ = fmt.Fprintln(w, "        directory containing workflow YAML files (default \".github/workflows\")")
	_, _ = fmt.Fprintln(w, "  -lockfile string")
	_, _ = fmt.Fprintln(w, "        path to workflow lock file (default \"workflow-lock.yaml\")")
	_, _ = fmt.Fprintln(w, "  -default-host string")
	_, _ = fmt.Fprintln(w, "        default host for plain owner/repo workflow refs (default \"github.com\")")
	_, _ = fmt.Fprintln(w, "  -format string")
	_, _ = fmt.Fprintln(w, "        output format for list and diff: text or json (default \"text\")")
	_, _ = fmt.Fprintln(w, "  -config string")
	_, _ = fmt.Fprintln(w, "        path to optional workflow-lock config file (default auto-detect .workflow-lock-config.yaml)")
}
