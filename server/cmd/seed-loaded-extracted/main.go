//go:build ignore

// One-shot smoke helper (not product seed): wipe feragame Domain and Commit jobs
// for /app dataReport. Run from this directory: go run main.go
// Stops after containerReg (lot stays open) and leaves a second intake at weightCheck
// so chromeDisplay can be printed. Focused SeedDemo(--replace) leaves one truck ×
// one lot through weigh + markProgress + containerReg.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
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
	domains := org.NewMongoDomainStore(db)
	configs := org.NewMongoConfigStore(db)
	if _, err := org.SeedDemo(ctx, domains, configs, org.SeedOptions{SkipEmptyDomainStaging: true}); err != nil {
		must(err)
	}
	scope := org.FeragameScope()
	must(domains.Put(ctx, scope, org.EmptyDomain()))

	c := &org.TaskCommitter{
		Domains:    domains,
		Configs:    configs,
		Authorizer: org.AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 10, 19, 16, 0, 0, 0, time.UTC) },
	}

	admin := org.OperatorIdentity{UserID: "living", Org: "feragame", Roles: []string{"administration", "logistic", "line"}}
	lineOp := org.OperatorIdentity{UserID: "living", Org: "feragame", Roles: []string{"line"}}
	logOp := org.OperatorIdentity{UserID: "living", Org: "feragame", Roles: []string{"logistic"}}

	arrive := commit(c, admin, org.CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    map[string]any{"truckWeight": 1000.0},
		Mint:     mintNew(),
	})
	truckID := arrive.JobInstanceID
	reg := commit(c, admin, org.CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value: map[string]any{
			"client": "CO", "wasteCode": "160214", "numeroFir": "FIR-SMOKE-1",
			"palletCount": 2.0, "truckWeightAfter": 700.0, "notes": "",
		},
	})
	loadID := firstLotID(reg.Domain)
	if loadID == "" {
		must(fmt.Errorf("expected minted lot"))
	}

	commit(c, logOp, org.CommitRequest{
		TaskPath:         "lot_processing/startLot",
		Value:            map[string]any{"lotId": loadID},
		Mint:             mintNew(),
		StreamInstanceID: truckID,
	})
	for _, row := range []struct {
		id     string
		weight float64
	}{
		{loadID + ".1", 11},
		{loadID + ".2", 22},
	} {
		commit(c, lineOp, org.CommitRequest{
			TaskPath:      "lot_processing/weigh",
			JobInstanceID: loadID,
			Value: map[string]any{
				"palletId":           row.id,
				"averagePanelWeight": 10.0,
				"tareKind":           "euro",
				"weight":             row.weight,
				"idoneita":           "NIR",
				"panelCount":         1.0,
			},
		})
	}
	commit(c, lineOp, org.CommitRequest{
		TaskPath:      "lot_processing/markProgress",
		JobInstanceID: loadID,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "done",
				loadID + ".2": "done",
			},
		},
	})
	res := commit(c, logOp, org.CommitRequest{
		TaskPath:      "lot_processing/containerReg",
		JobInstanceID: loadID,
		Value:         map[string]any{"material": "polvereDiVetro", "weight": 80.0},
	})
	res = commit(c, logOp, org.CommitRequest{
		TaskPath:      "lot_processing/containerReg",
		JobInstanceID: loadID,
		Value:         map[string]any{"material": "plastica", "weight": 40.0},
	})
	res = commit(c, logOp, org.CommitRequest{
		TaskPath:      "lot_processing/containerReg",
		JobInstanceID: loadID,
		Value:         map[string]any{"material": "alluminio", "weight": 25.0},
	})

	// Second intake parked at weightCheck (flat dataReport smoke).
	second := commit(c, admin, org.CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    map[string]any{"truckWeight": 800.0},
		Mint:     mintNew(),
	})
	secondTruck := second.JobInstanceID
	res = commit(c, admin, org.CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: secondTruck,
		Value: map[string]any{
			"client": "DA", "wasteCode": "160214", "numeroFir": "FIR-SMOKE-WC",
			"palletCount": 2.0, "truckWeightAfter": 500.0, "notes": "",
		},
	})

	ver, err := configs.ActiveVersion(ctx, org.FeragameScope())
	must(err)
	cfg, err := configs.Get(ctx, org.FeragameScope(), ver)
	must(err)
	domain := res.Domain
	wl := org.ProjectWorklistAll(cfg, domain, org.FixtureAllRolesOp())
	fmt.Printf("seeded feragame in %s\n", dbName)
	fmt.Printf("containers=%d submissions=%d\n", len(domain.Kind("container")), len(domain.Submissions))
	for _, j := range wl.Jobs {
		for _, inst := range j.Instances {
			fmt.Printf("%s@%s:\n", j.JobID, inst.ID)
			for _, tid := range j.TaskIDs {
				fmt.Printf("  %s = %s\n", tid, inst.Tasks[tid])
			}
		}
	}
	refs := org.CollectActionableTaskRefs(cfg, wl)
	var haveWC bool
	for _, ref := range refs {
		b, err := org.BuildTaskFormBundle(cfg, domain, ref)
		must(err)
		cd, _ := b.Bindings["chromeDisplay"].(map[string]any)
		if b.TaskID == "weightCheck" {
			disp, _ := cd["registeredData"].(map[string]any)
			rows, _ := disp["rows"].([]any)
			fmt.Printf("weightCheck@%s available; registeredData rows=%d\n", b.JobInstanceID, len(rows))
			haveWC = true
		}
	}
	fmt.Println("lot_processing stopped after containerReg (lot open); second intake at weightCheck")
	if !haveWC {
		must(fmt.Errorf("weightCheck bundle missing"))
	}
}

func commit(c *org.TaskCommitter, op org.OperatorIdentity, req org.CommitRequest) org.CommitResult {
	res, err := c.Commit(context.Background(), org.FeragameScope(), op, req)
	if err != nil {
		must(fmt.Errorf("%s: %w", req.TaskPath, err))
	}
	return res
}

func firstLotID(d org.Domain) string {
	for id := range d.Kind("lot") {
		return id
	}
	return ""
}

func mintNew() *org.CommitMint {
	return &org.CommitMint{Kind: org.MintKindStartingNew}
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
