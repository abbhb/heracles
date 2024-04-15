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
		config := viper.GetViper()

		compose, err := core.NewDockerCompose(config.GetString("exporter.compose_file"))
		if err != nil {
			log.Fatalf("failed to create docker compose: %+v", err)
		}

		exporter := core.NewDockerComposeExporter(
			compose,
			config.GetString("exporter.service"),
			config.GetDuration("exporter.startup_wait"),
		)

		var metrics []core.MetricsConfig
		err = config.UnmarshalKey("exporter.metrics", &metrics)
		if err != nil {
			log.Fatalf("failed to unmarshal metrics: %+v", err)
		}

		checker := core.NewMetricChecker(
			exporter,
			[]core.Fixture{compose},
			config.GetString("exporter.path"),
			config.GetStringSlice("exporter.disallowed_metrics"),
			config.GetBool("exporter.allow_empty"),
			metrics,
		)
		err = checker.Check(cmd.Context())

		switch eris.Cause(err) {
		case nil:
			log.Infof("metrics check passed!")
		case core.ErrCheck:
			log.Warnf("metrics check failed, %v", err)
		default:
			log.Fatalf("failed to run: %+v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	config := viper.GetViper()
	config.SetDefault("exporter.compose_file", "docker-compose.yml")
	config.SetDefault("exporter.service", "exporter")
	config.SetDefault("exporter.path", "/metrics")
	config.SetDefault("exporter.startup_wait", time.Second)
	config.SetDefault("exporter.allow_empty", false)
	config.SetDefault("exporter.disallowed_metrics", nil)
	config.SetDefault("exporter.metrics", nil)
}
