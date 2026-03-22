package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/owomeister/grotto/app"
)

var version = "dev"

func main() {
	noAI := flag.Bool("no-ai", false, "start without AI panel")
	aiProvider := flag.String("ai", "", "AI provider (kiro-cli, claude, codex)")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *showVersion {
		fmt.Println("grotto", version)
		os.Exit(0)
	}

	cfg := app.Config{
		NoAI:       *noAI,
		AIProvider: *aiProvider,
	}

	// Parse positional arg: file, dir, or file:line
	if flag.NArg() > 0 {
		arg := flag.Arg(0)
		if idx := strings.LastIndex(arg, ":"); idx > 0 {
			if line, err := strconv.Atoi(arg[idx+1:]); err == nil {
				cfg.Path = arg[:idx]
				cfg.Line = line
			} else {
				cfg.Path = arg
			}
		} else {
			cfg.Path = arg
		}
	}

	if cfg.Path == "" {
		cfg.Path = "."
	}

	p := tea.NewProgram(app.New(cfg))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
