package cmd

import (
	"time"

	"github.com/mrlyc/heracles/core"
	"github.com/mrlyc/heracles/log"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check exporter metrics",
	Run: func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()
		dockerCompose, _ := flags.GetString("docker-compose")
		config := viper.GetViper()

		compose, err := core.NewDockerCompose(
			dockerCompose,
			config.GetString("exporter.service"),
			config.GetDuration("exporter.duration"),
		)
		if err != nil {
			log.Fatalf("failed to create docker compose: %+v", err)
		}

		checker := core.NewMetricChecker(compose, []core.Fixture{compose}, config)
		err = checker.Check(cmd.Context())

		switch eris.Cause(err) {
		case core.ErrCheck:
			log.Warnf("metrics check failed: %v", err)
		default:
			log.Fatalf("failed to run: %+v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	flags := checkCmd.Flags()
	flags.StringP("docker-compose", "d", "docker-compose.yml", "Specify the docker-compose file")

	config := viper.GetViper()
	config.SetDefault("exporter.service", "exporter")
	config.SetDefault("exporter.path", "/metrics")
	config.SetDefault("exporter.duration", time.Second)
	config.SetDefault("exporter.allow_empty", false)
	config.SetDefault("exporter.disallowed_metrics", nil)
	config.SetDefault("exporter.metrics", nil)
}
