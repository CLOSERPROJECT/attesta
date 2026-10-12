package main

import (
	"context"
	"fmt"
	"strings"
)

func runPrepareCommand(ctx context.Context, args []string) error {
	policy, rest, err := parsePrepareArgs(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("prepare: unexpected arguments: %s", strings.Join(rest, " "))
	}

	deps := PrepareDeps{Policy: policy}

	return PrepareDemoStack(ctx, deps.withSeedRunner(func(ctx context.Context) error {
		return runSeedSession(ctx, nil, false)
	}))
}

func (d PrepareDeps) withSeedRunner(seed func(context.Context) error) PrepareDeps {
	d.Seed = seed
	return d
}

func parsePrepareArgs(args []string) (policy string, rest []string, err error) {
	policy = PreparePolicyCoolify
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help", arg == "-h":
			return "", nil, fmt.Errorf("usage: attesta prepare [--policy=coolify|worktree]")
		case arg == "--policy":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("prepare: --policy requires a value (coolify or worktree)")
			}
			i++
			normalized, nerr := normalizePreparePolicy(args[i])
			if nerr != nil {
				return "", nil, nerr
			}
			policy = normalized
		case strings.HasPrefix(arg, "--policy="):
			normalized, nerr := normalizePreparePolicy(strings.TrimPrefix(arg, "--policy="))
			if nerr != nil {
				return "", nil, nerr
			}
			policy = normalized
		case strings.HasPrefix(arg, "-"):
			return "", nil, fmt.Errorf("prepare: unknown flag %s", arg)
		default:
			rest = append(rest, arg)
		}
	}
	return policy, rest, nil
}
