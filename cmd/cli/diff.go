package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/cortea-ai/pg-migrant/internal/db"
	"github.com/cortea-ai/pg-migrant/internal/diffutils"
	"github.com/stripe/pg-schema-diff/pkg/diff"
	"github.com/stripe/pg-schema-diff/pkg/tempdb"
)

func Diff(ctx context.Context, conf *config.Config, migrate bool) error {
	if len(conf.GetSchemaFiles()) == 0 {
		return errors.New("no schema files provided")
	}

	dbConfig, err := conf.GetDBConfig()
	if err != nil {
		return err
	}

	tempDbFactory, err := tempdb.NewOnInstanceFactory(ctx,
		func(ctx context.Context, dbName string) (*sql.DB, error) {
			dbUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?search_path=public&sslmode=disable",
				dbConfig.User,
				dbConfig.Password,
				dbConfig.Host,
				dbConfig.Port,
				dbName, // we replace the db name
			)
			conn, err := db.NewConn(ctx, dbUrl)
			if err != nil {
				return nil, err
			}
			return conn.DB, nil
		},
		tempdb.WithRootDatabase(dbConfig.Database),
	)
	if err != nil {
		return err
	}
	defer func() {
		err := tempDbFactory.Close()
		if err != nil {
			fmt.Printf("error shutting down temp db factory: %v", err)
		}
	}()

	conn, _, err := db.NewConnEnsureVersionTable(ctx, conf.GetDBUrl())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	ddls, err := diffutils.GetDDLsFromFiles(conf.GetSchemaFiles())
	if err != nil {
		return err
	}
	schemaSource := diff.DDLSchemaSource(ddls)

	plan, err := diff.Generate(ctx, conn, schemaSource,
		diff.WithDataPackNewTables(),
		diff.WithExcludeSchemas(append(conf.GetExcludeSchemas(), db.PGMigrantSchema)...),
		diff.WithTempDbFactory(tempDbFactory),
	)
	if err != nil {
		return err
	}

	if len(plan.Statements) == 0 {
		println("schema matches expected. No plan generated")
		return nil
	}

	files, err := conf.GetMigrationFiles()
	if err != nil {
		return fmt.Errorf("reading migration directory: %w", err)
	}

	var maxVersion string
	var lastFile fs.DirEntry
	for _, file := range files {
		lastFile = file
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".sql") {
			continue
		}
		version, err := VersionFromFilename(name)
		if err != nil {
			continue
		}
		if version > maxVersion {
			maxVersion = version
		}
	}

	if lastFile != nil {
		lastFilePath := filepath.Join(conf.GetMigrationDir(), lastFile.Name())
		lastContent, err := os.ReadFile(lastFilePath)
		if err != nil {
			return fmt.Errorf("reading last migration file: %w", err)
		}
		if string(lastContent) == diffutils.PlanToPrettyS(plan) {
			println("No changes detected - migration content matches last file")
			return nil
		}
	}

	println(diffutils.PlanToPrettyS(plan))

	newVersionStr := "0000"
	if maxVersion != "" {
		newVersion, err := strconv.Atoi(maxVersion)
		if err != nil {
			return fmt.Errorf("invalid max version: %w", err)
		}
		newVersionStr = fmt.Sprintf("%04d", newVersion+1)
	}

	if migrate {
		if err := promptForApproval("Apply this migration?"); err != nil {
			return err
		}
		if err := conn.ApplyMigration(ctx, newVersionStr, diffutils.PlanToPrettyS(plan)); err != nil {
			return err
		}
		if conf.GetMigrationDir() == "" {
			return nil
		}
	}

	newFilePath := filepath.Join(conf.GetMigrationDir(), fmt.Sprintf("%s.sql", newVersionStr))
	if err := promptForApproval("Create new migration file?"); err != nil {
		return err
	}
	err = os.WriteFile(newFilePath, []byte(diffutils.PlanToPrettyS(plan)), 0644)
	if err != nil {
		return fmt.Errorf("writing migration file: %w", err)
	}

	fmt.Printf("\n✅ Created new migration file: %s\n", newFilePath)

	return nil
}
