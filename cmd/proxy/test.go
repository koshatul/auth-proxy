package main

import (
	"github.com/na4ma4/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// nolint: gochecknoglobals // cobra uses globals in main
var cmdTest = &cobra.Command{
	Use:    "test",
	Short:  "Test Command",
	Run:    testCommand,
	Hidden: true,
}

// nolint:gochecknoinits // init is used in main for cobra
func init() {
	rootCmd.AddCommand(cmdTest)
}

func testCommand(cmd *cobra.Command, args []string) {
	cfg := config.NewViperConfigFromViper(viper.GetViper(), "auth-proxy")

	logger, _ := cfg.ZapConfig().Build()
	defer logger.Sync() //nolint:errcheck

	hash := []byte("$2a$15$IA1JuaL5pCYoc1R4L5qJ/eXVrqcs13bjSHmbXQz7PmMiD.InITn3S")
	if err := bcrypt.CompareHashAndPassword(hash, []byte("password")); err != nil {
		logger.Panic("Compare Hash and Password Failed", zap.Error(err))
	} else {
		logger.Info("Compare Hash Passed")
	}
}
