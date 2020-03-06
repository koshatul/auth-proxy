package main

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/na4ma4/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// nolint: gochecknoglobals // cobra uses globals in main
var cmdMakePassword = &cobra.Command{
	Use:    "mkpasswd <username> [password]",
	Short:  "Generate a compatible hash for the legacy password",
	Run:    makePasswordCommand,
	Args:   cobra.MinimumNArgs(1),
	Hidden: true,
}

// nolint:gochecknoinits // init is used in main for cobra
func init() {
	rootCmd.AddCommand(cmdMakePassword)
}

// Added for future legacy support of bcrypted passwords
func makePasswordCommand(cmd *cobra.Command, args []string) {
	cfg := config.NewViperConfigFromViper(viper.GetViper(), "auth-proxy")

	logger, _ := cfg.ZapConfig().Build()
	defer logger.Sync() //nolint:errcheck

	username := args[0]
	password := ""

	if len(args) > 1 {
		// Password was specified on the command line
		password = args[1]
	} else {
		// Ask for password at prompt
		prompt := promptui.Prompt{
			Label: "Enter Password: ",
			Mask:  '*',
		}
		var err error

		if password, err = prompt.Run(); err != nil {
			logger.Panic("password entry failure", zap.Error(err))
		}
	}

	logger.Debug("username and password", zap.String("username", username), zap.String("password", password))

	pw, err := bcrypt.GenerateFromPassword([]byte(password), cfg.GetInt("auth.mincost"))
	if err != nil {
		logger.Panic("generate password failure", zap.Error(err))
	}

	logger.Debug("password returned", zap.ByteString("pw", pw))

	fmt.Printf("%s:%s\n", username, pw)
}
