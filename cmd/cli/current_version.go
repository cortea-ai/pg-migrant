package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/cortea-ai/pg-migrant/internal/db"
)

func CurrentVersion(ctx context.Context, conf *config.Config) error {
	conn, err := db.NewConn(ctx, conf.GetDBUrl())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	currentVersion, err := conn.CheckCurrentVersion(ctx)
	if err != nil {
		if errors.Is(err, db.ErrTableNotFound) {
			return fmt.Errorf("no migrations applied yet")
		}
		return err
	}
	if currentVersion != "" {
		println("Current version:", currentVersion)
	} else {
		println("No migrations applied yet")
	}
	return nil
}
