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
  --replace              Rewrite even when the empty-only marker org exists;
                         living Domain (+ passports) is wiped and rebuilt.
  --feragame-config      Active Config + wipe Domain (+ passports) to empty.
                         No Appwrite. Skips Formata. Always runs.
  --feragame-domain      Active Config + wipe and stage demo Domain.
                         No Appwrite. Skips Formata. Always runs.
  --feragame-identity    Feragame Appwrite hard reset + Active Config + wipe
                         Domain to empty. Skips Formata / orgs 1–3. Always runs.
  --feragame-full        Feragame Appwrite hard reset + Active Config + wipe
                         and stage demo Domain. Skips Formata / orgs 1–3.
                         Always runs.
  --file PATH            Taxonomy seed file (default: config/categories.yaml).
  --help, -h             Show this help.

Feragame flags are mutually exclusive. Every Feragame path always wipes Domain.

Examples:
  task seed
  task seed -- --replace
  task seed:feragame-config
  task seed:feragame-domain
  task seed:feragame-identity
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
	var found []living.SeedScope
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "--feragame-config":
			found = append(found, living.SeedScopeFeragameConfig)
		case "--feragame-domain":
			found = append(found, living.SeedScopeFeragameDomain)
		case "--feragame-identity":
			found = append(found, living.SeedScopeFeragameIdentity)
		case "--feragame-full":
			found = append(found, living.SeedScopeFeragameFull)
		}
	}
	if len(found) > 1 {
		return living.SeedScopeBoth, fmt.Errorf("seed: Feragame scope flags are mutually exclusive")
	}
	if len(found) == 1 {
		return found[0], nil
	}
	return living.SeedScopeBoth, nil
}
