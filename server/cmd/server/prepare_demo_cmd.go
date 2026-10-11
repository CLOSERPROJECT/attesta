package main

import (
	"context"
	"fmt"
	"log"
	"strings"
)

func runPrepareCommand(ctx context.Context, args []string) error {
	softBootstrap, softCredentialSkip, rest, err := parsePrepareArgs(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("prepare: unexpected arguments: %s", strings.Join(rest, " "))
	}

	deps := PrepareDeps{
		SoftBootstrap:      softBootstrap,
		SoftCredentialSkip: softCredentialSkip,
	}

	return PrepareDemoStack(ctx, deps.withSeedRunner(func(ctx context.Context) error {
		return withSeedSession(ctx, nil, workflowConfigDir(), openMongoTaxonomyStore, func() (IdentityStore, error) {
			return appwriteIdentityFromEnv(appwriteIdentityFromEnvOptions{RequireAll: true})
		}, func(sess seedSession) error {
			if err := Seed(ctx, SeedDeps{
				Identity:       sess.Identity,
				Store:          sess.Store,
				ConfigDir:      sess.ConfigDir,
				CategoriesPath: sess.CategoriesPath,
				Replace:        false,
			}); err != nil {
				return err
			}
			log.Printf("seed: demo fixture ready (empty-only)")
			return nil
		})
	}))
}

func (d PrepareDeps) withSeedRunner(seed func(context.Context) error) PrepareDeps {
	d.Seed = seed
	return d
}

func parsePrepareArgs(args []string) (softBootstrap, softCredentialSkip bool, rest []string, err error) {
	for _, arg := range args {
		switch arg {
		case "--soft-bootstrap":
			softBootstrap = true
		case "--soft-credential-skip":
			softCredentialSkip = true
		case "--help", "-h":
			return false, false, nil, fmt.Errorf("usage: attesta prepare [--soft-bootstrap] [--soft-credential-skip]")
		default:
			if strings.HasPrefix(arg, "-") {
				return false, false, nil, fmt.Errorf("prepare: unknown flag %s", arg)
			}
			rest = append(rest, arg)
		}
	}
	return softBootstrap, softCredentialSkip, rest, nil
}
