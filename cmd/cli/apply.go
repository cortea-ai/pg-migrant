package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/cortea-ai/pg-migrant/internal/db"
)

func Apply(ctx context.Context, conf *config.Config, autoApprove, dryRun bool) error {
	conn, err := db.NewConn(ctx, conf.GetDBUrl())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	currentVersion, err := conn.CheckCurrentVersion(ctx)
	if err != nil {
		if errors.Is(err, db.ErrTableNotFound) {
			if err = conn.CreateMigrationTable(ctx); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	versions, migrations, err := findPendingMigrations(currentVersion, conf.GetMigrationDir())
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		println("No pending migrations")
		return nil
	}
	for i, migration := range migrations {
		version := versions[i]
		println("Migration", version, "as", i+1, "of", len(migrations), "migrations:")
		println("\n---\n")
		println(migration)
		println("---\n")
		if !dryRun {
			if !autoApprove {
				if err := promptForApproval("Apply this migration?"); err != nil {
					return err
				}
			}
			if err := conn.ApplyMigration(ctx, version, migration); err != nil {
				return err
			}
		}
	}
	return nil
}

func promptForApproval(msg string) error {
	print(msg + " [y/N]: ")
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return err
	}
	if response != "y" && response != "Y" {
		return errors.New("migration aborted")
	}
	return nil
}
