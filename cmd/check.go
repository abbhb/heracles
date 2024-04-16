package cmd

import (
	"context"
	"time"

	"github.com/mrlyc/heracles/core"
	"github.com/mrlyc/heracles/log"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/dig"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check exporter metrics",
	Run: func(cmd *cobra.Command, args []string) {
		container := dig.New()
		for name, f := range map[string]interface{}{
			"context": cmd.Context,
			"flags":   cmd.Flags,
			"config": func(flags *pflag.FlagSet) *viper.Viper {
				group, _ := flags.GetString("group")
				root := viper.GetViper()
				config := root.Sub(group)
				if config == nil {
					log.Fatalf("invalid group: %s", group)
				}

				config.SetDefault("compose_file", "docker-compose.yml")
				config.SetDefault("service", "exporter")
				config.SetDefault("path", "/metrics")
				config.SetDefault("startup_wait", time.Second)
				config.SetDefault("allow_empty", false)
				config.SetDefault("disallowed_metrics", nil)
				config.SetDefault("metrics", nil)
				config.SetDefault("hooks", nil)

				return config
			},
			"docker-compose": func(config *viper.Viper, flags *pflag.FlagSet) (*core.DockerCompose, error) {
				removeAllImages, _ := flags.GetBool("remove-all-images")
				return core.NewDockerCompose(config.GetString("compose_file"), removeAllImages)
			},
			"exporter": func(config *viper.Viper, compose *core.DockerCompose) core.Exporter {
				return core.NewDockerComposeExporter(
					compose,
					config.GetString("service"),
					config.GetDuration("startup_wait"),
				)
			},
			"metrics-config": func(config *viper.Viper) ([]core.MetricsConfig, error) {
				var metrics []core.MetricsConfig
				err := config.UnmarshalKey("exporter.metrics", &metrics)
				return metrics, eris.Wrap(err, "metrics-config unmarshaling failed")
			},
			"script-fixtures": func(config *viper.Viper) ([]core.ScriptFixture, error) {
				var scriptFixtures []core.ScriptFixture
				err := config.UnmarshalKey("exporter.hooks", &scriptFixtures)
				return scriptFixtures, eris.Wrap(err, "script-fixtures unmarshaling failed")
			},
			"fixtures": func(compose *core.DockerCompose, scriptFixtures []core.ScriptFixture) []core.Fixture {
				fixtures := []core.Fixture{compose}
				for _, fixture := range scriptFixtures {
					fixtures = append(fixtures, fixture)
				}

				return fixtures
			},
			"metric-checker": func(exporter core.Exporter, fixtures []core.Fixture, config *viper.Viper, metrics []core.MetricsConfig) *core.MetricChecker {
				return core.NewMetricChecker(
					exporter,
					fixtures,
					config.GetString("path"),
					config.GetStringSlice("disallowed_metrics"),
					config.GetBool("allow_empty"),
					metrics,
				)
			},
		} {
			err := container.Provide(f)
			if err != nil {
				log.Fatalf("failed to provide %s: %v", name, err)
			}
		}

		err := container.Invoke(func(ctx context.Context, checker *core.MetricChecker) error {
			return checker.Check(ctx)
		})

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

	flags := checkCmd.Flags()
	flags.StringP("group", "g", "exporter", "config group")
	flags.Bool("remove-all-images", false, "remove all images after check")
}
