// One-shot: wipe feragame Domain and Commit smoke jobs for /app dataReport:
//   - intake NIR-…-NO-1 through weightCheck + line through containerReg → loadedExtracted
//   - intake NIR-…-SU-1 through palletWeight only → weightCheck available
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
	if _, err := living.SeedDemo(ctx, domains, configs, living.SeedOptions{WipeDomain: true}); err != nil {
		must(err)
	}

	c := &living.Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: living.AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC) },
	}

	intakeID := "NIR-261019-NO-1"
	admin := living.OperatorIdentity{UserID: "living", Org: "feragame", Roles: []string{"administration", "logistic", "line"}}
	lineOp := living.OperatorIdentity{UserID: "living", Org: "feragame", Roles: []string{"line"}}
	logOp := living.OperatorIdentity{UserID: "living", Org: "feragame", Roles: []string{"logistic"}}

	commit(c, admin, living.CommitRequest{
		TaskPath: "intake/register", JobInstanceID: intakeID,
		Value: map[string]any{
			"client": "NO", "date": "2026-10-19", "wasteCode": "191212",
			"idoneitaRiutilizzo": "NIR", "truckWeight": 1000.0, "truckTare": 200.0,
			"palletCount": 2.0, "notes": "",
		},
		Mint: mintNew(),
	})
	commit(c, admin, living.CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: intakeID,
		Value: map[string]any{"codiceFir": "FIR-SMOKE-1"},
	})
	commit(c, logOp, living.CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: intakeID,
		Value: map[string]any{
			"averagePanelWeight": 10.0,
			"tareKind":           "euro",
			"weights": map[string]any{
				intakeID + ".1": map[string]any{"weight": 125.0, "tareKind": "euro", "idoneita": "NIR", "panelCount": 10.0},
				intakeID + ".2": map[string]any{"weight": 100.0, "tareKind": "euro", "idoneita": "NIR", "panelCount": 7.0},
			},
		},
	})
	commit(c, admin, living.CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: intakeID,
		Value: map[string]any{"checkedTruckWeight": 1000.0, "checkedPalletSum": 225.0},
	})
	commit(c, lineOp, living.CommitRequest{
		TaskPath: "line/shift", JobInstanceID: "1st",
		Value: map[string]any{"shift": "1st"},
		Mint:  mintNew(),
	})
	commit(c, lineOp, living.CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: "1st",
		Value: map[string]any{
			"progress": map[string]any{
				intakeID + ".1": "done",
				intakeID + ".2": "done",
			},
		},
	})
	res := commit(c, logOp, living.CommitRequest{
		TaskPath: "line/containerReg", JobInstanceID: "1st",
		Value: map[string]any{
			"sourceLoadId": intakeID,
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
			"client": "SU", "date": "2026-10-19", "wasteCode": "191212",
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
	wl := living.ProjectWorklist(cfg, res.Domain)
	fmt.Printf("seeded feragame in %s\n", dbName)
	fmt.Printf("containers=%d submissions=%d\n", len(res.Domain.Container), len(res.Domain.Submissions))
	for _, j := range wl.Jobs {
		for _, inst := range j.Instances {
			fmt.Printf("%s@%s:\n", j.JobID, inst.ID)
			for _, tid := range j.TaskIDs {
				fmt.Printf("  %s = %s\n", tid, inst.Tasks[tid])
			}
		}
	}
	bundles, err := living.ProjectAvailableBundles(cfg, res.Domain, wl)
	must(err)
	var haveLE, haveWC bool
	for _, b := range bundles {
		cd, _ := b.Bindings["chromeDisplay"].(map[string]any)
		switch b.TaskID {
		case "loadedExtracted":
			disp, _ := cd["resultsReport"].(map[string]any)
			secs, _ := disp["sections"].([]any)
			fmt.Printf("loadedExtracted available; resultsReport sections=%d\n", len(secs))
			haveLE = true
		case "weightCheck":
			disp, _ := cd["registeredData"].(map[string]any)
			rows, _ := disp["rows"].([]any)
			fmt.Printf("weightCheck@%s available; registeredData rows=%d\n", b.JobInstanceID, len(rows))
			haveWC = true
		}
	}
	if !haveLE {
		must(fmt.Errorf("loadedExtracted bundle missing"))
	}
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
