package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cortea-ai/pg-migrant/cmd/cli"
	"github.com/cortea-ai/pg-migrant/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const pgMigrant = "pg-migrant"

var (
	rootCmd = &cobra.Command{
		Use:          pgMigrant,
		Short:        "A cli utility for db migrations",
		SilenceUsage: true,
	}
	configPath string
	env        string
	vars       = make(config.Vars)
)

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.AddCommand(currentVersionCmd())
	rootCmd.AddCommand(diffCmd())
	rootCmd.AddCommand(applyCmd())
	rootCmd.AddCommand(pendingMigrationsCmd())
	rootCmd.AddCommand(checkCmd())
	rootCmd.AddCommand(squashCmd())
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		// On first signal seen, cancel the context. On the second signal, force stop immediately.
		stop := make(chan os.Signal, 2)
		defer close(stop)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		defer signal.Stop(stop)
		<-stop   // wait for first interrupt
		cancel() // cancel context to gracefully stop
		rootCmd.Println("interrupt received, wait for exit or ^C to terminate")
		// Wait for the context to be canceled. Issuing a second interrupt will cause the process to force stop.
		<-stop // will not block if no signal received due to main routine exiting
		os.Exit(1)
	}()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func addGlobalFlags(set *pflag.FlagSet) {
	set.StringVar(&env, "env", "", "set which env to use from the config file")
	set.Var(&vars, "var", "input variables")
	set.StringVarP(&configPath, "config", "c", "./"+pgMigrant+".hcl", "Path to the configuration file")
}

func currentVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current-version",
		Short: "Get the current migration version of the db",
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.GetConfig(configPath, env, vars)
			if err != nil {
				return err
			}
			return cli.CurrentVersion(cmd.Context(), conf)
		},
	}
	addGlobalFlags(cmd.PersistentFlags())
	return cmd
}

func diffCmd() *cobra.Command {
	var (
		migrate = "migrate"
	)
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Diff the current schema against the db",
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.GetConfig(configPath, env, vars)
			if err != nil {
				return err
			}
			migrate, err := cmd.Flags().GetBool(migrate)
			if err != nil {
				return err
			}
			return cli.Diff(cmd.Context(), conf, migrate)
		},
	}
	addGlobalFlags(cmd.PersistentFlags())
	cmd.Flags().Bool(migrate, false, "Run diffed migrations on the fly")
	return cmd
}

func applyCmd() *cobra.Command {
	var (
		autoApprove = "auto-approve"
		dryRun      = "dry-run"
	)
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.GetConfig(configPath, env, vars)
			if err != nil {
				return err
			}
			autoApprove, err := cmd.Flags().GetBool(autoApprove)
			if err != nil {
				return err
			}
			dryRun, err := cmd.Flags().GetBool(dryRun)
			if err != nil {
				return err
			}
			return cli.Apply(cmd.Context(), conf, autoApprove, dryRun)
		},
	}
	addGlobalFlags(cmd.PersistentFlags())
	cmd.Flags().Bool(autoApprove, false, "Automatically approve migrations")
	cmd.Flags().Bool(dryRun, false, "Simulate the migration without applying changes")
	return cmd
}

func pendingMigrationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pending-migrations",
		Short: "Print the version for each pending migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.GetConfig(configPath, env, vars)
			if err != nil {
				return err
			}
			return cli.PendingMigrations(cmd.Context(), conf)
		},
	}
	addGlobalFlags(cmd.PersistentFlags())
	return cmd
}

func checkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check need for rebasing and no gaps in version numbering. Requires GITHUB_TOKEN.",
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.GetConfig(configPath, env, vars)
			if err != nil {
				return err
			}
			token, ok := os.LookupEnv("GITHUB_TOKEN")
			if !ok {
				return fmt.Errorf("GITHUB_TOKEN is not set")
			}
			return cli.Check(cmd.Context(), conf, token)
		},
	}
	addGlobalFlags(cmd.PersistentFlags())
	return cmd
}

func squashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "squash",
		Short: "Squash pending migrations into a single migration. Requires GITHUB_TOKEN.",
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.GetConfig(configPath, env, vars)
			if err != nil {
				return err
			}
			token, ok := os.LookupEnv("GITHUB_TOKEN")
			if !ok {
				return fmt.Errorf("GITHUB_TOKEN is not set")
			}
			return cli.Squash(cmd.Context(), conf, token)
		},
	}
	addGlobalFlags(cmd.PersistentFlags())
	return cmd
}
