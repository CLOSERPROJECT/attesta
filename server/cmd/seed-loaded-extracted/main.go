//go:build ignore

// One-shot smoke helper (not product seed): wipe feragame Domain and Commit jobs
// for /app dataReport. Run from this directory: go run main.go
// loadedExtracted is parked off the live line (ticket 02); this seed stops at
// containerReg/register and still prints weightCheck chrome when available.
// Focused SeedDemo(--replace) leaves one truck × two loads + one line turn through palletWeight.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/living"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	uri := env("MONGODB_URI", "mongodb://localhost:27017")
	dbName := env("MONGODB_DATABASE", "closer_demo")
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	must(err)
	defer client.Disconnect(ctx)
	must(client.Ping(ctx, nil))

	db := client.Database(dbName)
	domains := living.NewMongoDomainStore(db)
	configs := living.NewMongoConfigStore(db)
	// Active Config only — smoke builds Domain from empty; staged matrix would collide on line "1st".
	if _, err := living.SeedDemo(ctx, domains, configs, living.SeedOptions{SkipEmptyDomainStaging: true}); err != nil {
		must(err)
	}
	must(domains.Put(ctx, "feragame", living.EmptyDomain()))

	c := &living.Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: living.AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC) },
	}

	truckID := living.DemoTruckInstanceID
	loadID := living.DemoLoadInstanceID
	admin := living.OperatorIdentity{UserID: "living", Org: "feragame", Roles: []string{"administration", "logistic", "line"}}
	lineOp := living.OperatorIdentity{UserID: "living", Org: "feragame", Roles: []string{"line"}}
	logOp := living.OperatorIdentity{UserID: "living", Org: "feragame", Roles: []string{"logistic"}}

	commit(c, admin, living.CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{"checkedTruckWeight": 100.0, "checkedPalletSum": 33.0},
	})
	commit(c, lineOp, living.CommitRequest{
		TaskPath: living.GroupTaskPath("line", "lineLoad", living.ProvisionalGroupInstanceID, "mark"),
		JobInstanceID: living.DemoLineShiftID,
		Mint: &living.CommitMint{Kind: living.MintKindSpawningNew},
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "done",
				loadID + ".2": "done",
			},
		},
	})
	must(func() error {
		_, err := c.CloseGroup(ctx, "feragame", lineOp, living.CloseGroupRequest{
			JobID: "line", JobInstanceID: living.DemoLineShiftID, GroupID: "lineLoad",
		})
		return err
	}())
	res := commit(c, logOp, living.CommitRequest{
		TaskPath: living.GroupTaskPath("line", "containerReg", living.ProvisionalGroupInstanceID, "register"),
		JobInstanceID: living.DemoLineShiftID,
		Mint: &living.CommitMint{Kind: living.MintKindSpawningNew},
		Value: map[string]any{
			"sourceLoadId": loadID,
			"polvereDiVetro": []any{
				map[string]any{"weight": 80.0},
			},
			"plastica": []any{
				map[string]any{"weight": 40.0},
			},
			"alluminio": []any{
				map[string]any{"weight": 25.0},
			},
		},
	})

	// Second intake parked at weightCheck (flat dataReport smoke).
	weightCheckID := "NIR-261019-SU-1"
	commit(c, admin, living.CommitRequest{
		TaskPath: "intake/register", JobInstanceID: weightCheckID,
		Value: map[string]any{
			"client": "DA", "date": "2026-10-19", "wasteCode": "160214",
			"idoneitaRiutilizzo": "NIR", "truckWeight": 800.0, "truckTare": 180.0,
			"palletCount": 2.0, "notes": "",
		},
		Mint: mintNew(),
	})
	commit(c, admin, living.CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: weightCheckID,
		Value: map[string]any{"codiceFir": "FIR-SMOKE-WC"},
	})
	res = commit(c, logOp, living.CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: weightCheckID,
		Value: map[string]any{
			"averagePanelWeight": 10.0,
			"tareKind":           "euro",
			"weights": map[string]any{
				weightCheckID + ".1": map[string]any{"weight": 110.0, "tareKind": "euro", "idoneita": "NIR", "panelCount": 8.0},
				weightCheckID + ".2": map[string]any{"weight": 95.0, "tareKind": "euro", "idoneita": "NIR", "panelCount": 7.0},
			},
		},
	})

	ver, err := configs.ActiveVersion(ctx, "feragame")
	must(err)
	cfg, err := configs.Get(ctx, "feragame", ver)
	must(err)
	wl := living.ProjectWorklist(cfg, res.Domain, living.FixtureAllRolesOp())
	fmt.Printf("seeded feragame in %s\n", dbName)
	fmt.Printf("containers=%d submissions=%d\n", len(res.Domain.Kind("container")), len(res.Domain.Submissions))
	for _, j := range wl.Jobs {
		for _, inst := range j.Instances {
			fmt.Printf("%s@%s:\n", j.JobID, inst.ID)
			for _, tid := range j.TaskIDs {
				fmt.Printf("  %s = %s\n", tid, inst.Tasks[tid])
			}
		}
	}
	refs := living.CollectActionableTaskRefs(cfg, wl)
	var haveWC bool
	for _, ref := range refs {
		b, err := living.BuildTaskFormBundle(cfg, res.Domain, ref)
		must(err)
		cd, _ := b.Bindings["chromeDisplay"].(map[string]any)
		if b.TaskID == "weightCheck" {
			disp, _ := cd["registeredData"].(map[string]any)
			rows, _ := disp["rows"].([]any)
			fmt.Printf("weightCheck@%s available; registeredData rows=%d\n", b.JobInstanceID, len(rows))
			haveWC = true
		}
	}
	fmt.Println("loadedExtracted parked off live line (fixtures/parked-loaded-extracted.yaml)")
	if !haveWC {
		must(fmt.Errorf("weightCheck bundle missing"))
	}
}

func commit(c *living.Committer, op living.OperatorIdentity, req living.CommitRequest) living.CommitResult {
	res, err := c.Commit(context.Background(), "feragame", op, req)
	if err != nil {
		must(fmt.Errorf("%s: %w", req.TaskPath, err))
	}
	return res
}

func mintNew() *living.CommitMint {
	return &living.CommitMint{Kind: living.MintKindStartingNew}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
