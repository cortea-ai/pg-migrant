package cli

import (
	"context"
	"fmt"

	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/cortea-ai/pg-migrant/internal/db"
)

func Clean(ctx context.Context, conf *config.Config) error {
	if !conf.GetAllowDBClean() {
		return fmt.Errorf("allow_db_clean=false, refusing to clean database schema")
	}
	conn, err := db.NewConn(ctx, conf.GetDBUrl())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if err := promptForApproval("Clean database schema?"); err != nil {
		return err
	}
	if err := conn.CleanSchema(ctx); err != nil {
		return err
	}
	fmt.Printf("\n✅ Cleaned database schema\n")
	return nil
}
