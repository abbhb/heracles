package cmd

import (
	"time"

	"github.com/mrlyc/heracles/core"
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
			panic(err)
		}

		runner := core.NewRunner(compose, []core.Fixture{compose}, config)
		err = runner.Run(cmd.Context())
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	flags := checkCmd.Flags()
	flags.StringP("docker-compose", "d", "docker-compose.yml", "Specify the docker-compose file")
	checkCmd.MarkFlagRequired("docker-compose")

	config := viper.GetViper()
	config.SetDefault("exporter.service", "exporter")
	config.SetDefault("exporter.duration", time.Second)
}
