package cmd

import (
	"context"
	"os"
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
				config.SetDefault("container", "exporter")
				config.SetDefault("base_url", "")
				config.SetDefault("path", "/metrics")
				config.SetDefault("wait", 3*time.Second)
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
				if config.GetString("base_url") == "" {
					return core.NewDockerComposeExporter(
						compose,
						config.GetString("container"),
						config.GetDuration("wait"),
					)
				} else {
					return core.NewExternalExporter(config.GetString("base_url"))
				}
			},
			"metrics-config": func(config *viper.Viper) ([]core.MetricsConfig, error) {
				var metrics []core.MetricsConfig
				err := config.UnmarshalKey("metrics", &metrics)
				return metrics, eris.Wrap(err, "metrics-config unmarshaling failed")
			},
			"fixtures": func(compose *core.DockerCompose, config *viper.Viper) []core.Fixture {
				fixtures := []core.Fixture{compose}

				var hooks []core.ScriptHook
				err := config.UnmarshalKey("hooks", &hooks)
				if err != nil {
					log.Warnf("hooks unmarshaling failed: %v", err)
				}

				for _, hook := range hooks {
					if hook.Container == "" {
						fixtures = append(fixtures, core.NewScriptFixture(
							hook.Name,
							hook.Setup,
							hook.TearDown,
						))
					} else {
						fixtures = append(fixtures, core.NewContainerScriptFixture(
							compose,
							hook.Name,
							hook.Container,
							hook.Setup,
							hook.TearDown,
						))
					}
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
					config.GetDuration("wait"),
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
			log.Errorf("metrics check failed")
			os.Exit(1)
		default:
			log.Fatalf("crashed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	flags := checkCmd.Flags()
	flags.StringP("group", "g", "exporter", "config group")
	flags.Bool("remove-all-images", false, "remove all images after check")
}
