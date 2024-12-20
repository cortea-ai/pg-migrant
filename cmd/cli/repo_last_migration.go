package cli

import (
	"context"

	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func RepoLastMigration(ctx context.Context, conf *config.Config, token string) error {
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))

	ghClient := github.NewClient(tc)

	_, migrations, _, err := ghClient.Repositories.GetContents(
		ctx,
		conf.GetGitHubConfig().Owner,
		conf.GetGitHubConfig().Repo,
		conf.GetMigrationDir(),
		nil,
	)
	if err != nil {
		return err
	}

	// Get latest version from remote migrations
	var currentVersion string
	if len(migrations) > 0 {
		lastMigration := migrations[len(migrations)-1]
		currentVersion, err = VersionFromFilename(lastMigration.GetName())
		if err != nil {
			return err
		}
	}
	if currentVersion != "" {
		println("Current version:", currentVersion)
	} else {
		println("No migrations applied yet")
	}
	return nil
}
