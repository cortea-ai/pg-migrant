package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func Squash(ctx context.Context, conf *config.Config, token string) error {
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

	_, pendingMigrations, err := findPendingMigrations(currentVersion, conf.GetMigrationDir())
	if err != nil {
		return err
	}

	if len(pendingMigrations) == 0 {
		return nil
	}

	// Get all migration files after current version
	files, err := os.ReadDir(conf.GetMigrationDir())
	if err != nil {
		return err
	}

	var pendingFiles []string
	var firstFile string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		version, err := VersionFromFilename(file.Name())
		if err != nil {
			return err
		}
		if version > currentVersion {
			if firstFile == "" {
				firstFile = file.Name()
			}
			pendingFiles = append(pendingFiles, file.Name())
		}
	}

	// Combine all migrations into one file
	var combinedMigration string
	for _, content := range pendingMigrations {
		combinedMigration += content + "\n"
	}

	// Write combined migration to first file
	firstFilePath := filepath.Join(conf.GetMigrationDir(), firstFile)
	if err := os.WriteFile(firstFilePath, []byte(combinedMigration), 0644); err != nil {
		return err
	}

	// Delete other pending migration files
	for _, file := range pendingFiles {
		if file == firstFile {
			continue
		}
		if err := os.Remove(filepath.Join(conf.GetMigrationDir(), file)); err != nil {
			return err
		}
	}

	println("✅ Squashed migrations")

	return nil
}
