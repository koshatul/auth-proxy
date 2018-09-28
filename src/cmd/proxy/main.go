package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use: "proxy",
}

func init() {
	cobra.OnInitialize(configInit)

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug output")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindEnv("debug", "DEBUG")

	rootCmd.PersistentFlags().BoolP("info", "i", false, "Info output (includes non-apache format output)")
	viper.BindPFlag("extra-logs", rootCmd.PersistentFlags().Lookup("info"))
	viper.BindEnv("extra-logs", "EXTRA_LOGS")
}

func main() {
	rootCmd.Execute()
}
