package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/cortea-ai/pg-migrant/internal/db"
)

func PendingMigrations(ctx context.Context, conf *config.Config) error {
	conn, currentVersion, err := db.NewConnEnsureVersionTable(ctx, conf.GetDBUrl())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	versions, _, err := findPendingMigrations(currentVersion, conf.GetMigrationDir())
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		println("No pending migrations")
		return nil
	}
	println("Pending migrations:")
	for _, version := range versions {
		println(">", version)
	}
	return nil
}

func findPendingMigrations(currentVersion string, migrationDir string) ([]string, []string, error) {
	files, err := os.ReadDir(migrationDir)
	if err != nil {
		return nil, nil, err
	}
	var versions []string
	var migrations []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		version, err := VersionFromFilename(file.Name())
		if err != nil {
			return nil, nil, err
		}
		if version > currentVersion {
			content, err := os.ReadFile(filepath.Join(migrationDir, file.Name()))
			if err != nil {
				return nil, nil, err
			}
			versions = append(versions, version)
			migrations = append(migrations, string(content))
		}
	}
	return versions, migrations, nil
}

func VersionFromFilename(filename string) (string, error) {
	filenameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(filenameWithoutExt, "_")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid filename: %s", filename)
	}
	if err := ValidateVersion(parts[0]); err != nil {
		return "", err
	}
	return parts[0], nil
}

func ValidateVersion(version string) error {
	if len(version) != 4 {
		return fmt.Errorf("version must be 4 characters long: %s", version)
	}
	_, err := strconv.Atoi(version)
	if err != nil {
		return fmt.Errorf("version must be numeric: %s", version)
	}
	return nil
}
