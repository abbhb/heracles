package cmd

import (
	"fmt"
	"os"

	"github.com/mrlyc/heracles/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/testcontainers/testcontainers-go"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// initConfig 读取配置文件
func initConfig() {
	if cfgFile != "" {
		// 使用 flag 指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 寻找 home 目录.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// 在 home 目录中搜索名为 ".myapp" 的配置
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".myapp")
	}

	// 读取匹配环境变量
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func initLogger() {
	log.UpdateDefaultLogger()
	testcontainers.Logger = log.GetDefaultLogger()
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	flags := rootCmd.PersistentFlags()
	flags.StringVarP(&cfgFile, "config", "c", ".heracles.yaml", "config file (default is .heracles.yaml)")
	flags.StringP("log-level", "l", "info", "log level")

	_ = viper.BindPFlag("log_level", flags.Lookup("log-level"))
}
