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
  --config-only      Living Active Config only (Feragame fixture). Does not
                     wipe or stage Domain. Skips identity/taxonomy/catalog.
  --domain-only      Wipe and rebuild staged Domain against existing Active
                     Config (+ passports). Fails if Active Config is missing.
                     Skips identity/taxonomy/catalog. Alias: --data-only.
  --file PATH        Taxonomy seed file (default: config/categories.yaml).
  --help, -h         Show this help.

Living scope defaults to both (config + domain). --config-only and
--domain-only/--data-only are mutually exclusive.

Examples:
  task seed
  task seed -- --replace
  task seed -- --config-only
  task seed -- --domain-only
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
	configOnly := false
	domainOnly := false
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "--config-only":
			configOnly = true
		case "--domain-only", "--data-only":
			domainOnly = true
		}
	}
	if configOnly && domainOnly {
		return living.SeedScopeBoth, fmt.Errorf("seed: --config-only and --domain-only/--data-only are mutually exclusive")
	}
	if configOnly {
		return living.SeedScopeConfig, nil
	}
	if domainOnly {
		return living.SeedScopeDomain, nil
	}
	return living.SeedScopeBoth, nil
}
