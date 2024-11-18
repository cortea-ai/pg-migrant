package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Migration struct {
	Filename string
	Version  string
	Content  string
}

func Check(ctx context.Context, conf *config.Config, token string) error {
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))

	ghClient := github.NewClient(tc)

	// Get remote migrations
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

	// Get local migrations
	localFiles, err := os.ReadDir(conf.GetMigrationDir())
	if err != nil {
		return fmt.Errorf("failed to read local migration directory: %w", err)
	}
	localMigrations := make([]Migration, 0)
	for _, file := range localFiles {
		if file.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(conf.GetMigrationDir(), file.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}
		version, err := VersionFromFilename(file.Name())
		if err != nil {
			return err
		}
		localMigrations = append(localMigrations, Migration{
			Filename: file.Name(),
			Version:  version,
			Content:  string(content),
		})
	}

	// Ensure no gaps in migration versions
	var prevVersion int
	for _, m := range localMigrations {
		version, err := strconv.Atoi(m.Version)
		if err != nil {
			return fmt.Errorf("failed to parse version %s as number: %w", m.Version, err)
		}
		if prevVersion != 0 && version != prevVersion+1 {
			return fmt.Errorf("migration versions must increment by 1, but got %d after %d", version, prevVersion)
		}
		prevVersion = version
	}

	for i, m := range migrations {
		println("Checking remote migration:", m.GetName())
		localM := localMigrations[i]
		if m.GetName() != localM.Filename {
			return fmt.Errorf("migration %s exists in remote but not locally", m.GetName())
		}
		version, err := VersionFromFilename(m.GetName())
		if err != nil {
			return err
		}
		if localM.Version != version {
			return fmt.Errorf("migration %s has different version locally than in remote", localM.Filename)
		}
		content, err := getFileContent(ctx, ghClient, conf, *m.Path)
		if err != nil {
			return fmt.Errorf("failed to get remote content for %s: %w", m.GetName(), err)
		}
		if localM.Content != content {
			return fmt.Errorf("migration %s has different content locally than in remote", localM.Filename)
		}
	}

	println("\n✅ All migrations are in sync\n")

	return nil
}

func getFileContent(ctx context.Context, ghClient *github.Client, conf *config.Config, path string) (string, error) {
	rc, err := ghClient.Repositories.DownloadContents(ctx,
		conf.GetGitHubConfig().Owner,
		conf.GetGitHubConfig().Repo,
		path,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to download content: %w", err)
	}
	defer rc.Close()
	contentBytes, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}
	return string(contentBytes), nil
}
