package main

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var cmdTest = &cobra.Command{
	Use:    "test",
	Short:  "Test Command",
	Run:    testCommand,
	Hidden: true,
}

func init() {
	rootCmd.AddCommand(cmdTest)
}

func testCommand(cmd *cobra.Command, args []string) {
	cfg := zapConfig()
	logger, _ := cfg.Build()
	defer logger.Sync()

	hash := []byte("$2a$15$IA1JuaL5pCYoc1R4L5qJ/eXVrqcs13bjSHmbXQz7PmMiD.InITn3S")
	err := bcrypt.CompareHashAndPassword(hash, []byte("password"))
	if err != nil {
		logger.Panic("Compare Hash and Password Failed", zap.Error(err))
	} else {
		logger.Info("Compare Hash Passed")
	}
}
