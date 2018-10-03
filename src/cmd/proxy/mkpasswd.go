package main

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var cmdMakePassword = &cobra.Command{
	Use:    "mkpasswd <username> [password]",
	Short:  "Generate a compatible hash for the legacy password",
	Run:    makePasswordCommand,
	Args:   cobra.MinimumNArgs(1),
	Hidden: true,
}

func init() {
	rootCmd.AddCommand(cmdMakePassword)
}

// Added for future legacy support of bcrypted passwords
func makePasswordCommand(cmd *cobra.Command, args []string) {
	cfg := zapConfig()
	logger, _ := cfg.Build()
	defer logger.Sync()

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
		password, err = prompt.Run()
		if err != nil {
			logger.Panic("Password entry failure", zap.Error(err))
		}
	}

	logger.Debug("Username and Password", zap.String("username", username), zap.String("password", password))

	pw, err := bcrypt.GenerateFromPassword([]byte(password), viper.GetInt("auth.mincost"))
	if err != nil {
		logger.Panic("Generate password failure", zap.Error(err))
	}

	logger.Debug("Password Returned", zap.ByteString("pw", pw))

	fmt.Printf("%s:%s\n", username, pw)
}
