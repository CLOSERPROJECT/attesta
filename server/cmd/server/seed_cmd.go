package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/CLOSERPROJECT/attesta/server/internal/living"
)

const seedUsage = `Usage: attesta seed [flags]

Demo fixture seed (identity + taxonomy + catalog + living by default).

Flags:
  --replace          Rewrite even when the empty-only marker org exists;
                     living Domain (+ passports) is wiped and rebuilt.
  --feragame-core    Hard-reset Feragame only: Appwrite org/users, Active
                     Config, wipe Domain (+ passports) to empty. No staged
                     demo plant. Skips Formata and orgs 1–3. Always runs.
  --feragame-full    Same as --feragame-core, then restage the demo Domain
                     plant. Always runs.
  --file PATH        Taxonomy seed file (default: config/categories.yaml).
  --help, -h         Show this help.

--feragame-core and --feragame-full are mutually exclusive.

Examples:
  task seed
  task seed -- --replace
  task seed:feragame-core
  task seed:feragame-full
`

func runSeedCommand(ctx context.Context, args []string) error {
	if seedHelpFromArgs(args) {
		fmt.Print(seedUsage)
		return nil
	}
	scope, err := seedLivingScopeFromArgs(args)
	if err != nil {
		return err
	}
	return runSeedSession(ctx, args, seedReplaceFromArgs(args), scope, nil)
}

func seedReplaceFromArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--replace" {
			return true
		}
	}
	return false
}

func seedHelpFromArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func seedLivingScopeFromArgs(args []string) (living.SeedScope, error) {
	core := false
	full := false
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "--feragame-core":
			core = true
		case "--feragame-full":
			full = true
		}
	}
	if core && full {
		return living.SeedScopeBoth, fmt.Errorf("seed: --feragame-core and --feragame-full are mutually exclusive")
	}
	if core {
		return living.SeedScopeFeragameCore, nil
	}
	if full {
		return living.SeedScopeFeragameFull, nil
	}
	return living.SeedScopeBoth, nil
}
