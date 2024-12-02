package cli

import (
	"context"

	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/cortea-ai/pg-migrant/internal/db"
)

func CurrentVersion(ctx context.Context, conf *config.Config) error {
	conn, currentVersion, err := db.NewConnEnsureVersionTable(ctx, conf.GetDBUrl())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	println("Current version:", currentVersion)
	return nil
}
