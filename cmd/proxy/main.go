package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// nolint: gochecknoglobals // cobra uses globals in main
var rootCmd = &cobra.Command{
	Use: "proxy",
}

// nolint:gochecknoinits // init is used in main for cobra
func init() {
	cobra.OnInitialize(configInit)

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug output")
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindEnv("debug", "DEBUG")

	rootCmd.PersistentFlags().BoolP("info", "i", false, "Info output (includes non-apache format output)")
	_ = viper.BindPFlag("extra-logs", rootCmd.PersistentFlags().Lookup("info"))
	_ = viper.BindEnv("extra-logs", "EXTRA_LOGS")
}

func main() {
	_ = rootCmd.Execute()
}
