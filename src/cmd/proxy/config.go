package main

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func configDefaults() {
	viper.SetDefault("server.address", "0.0.0.0")
	viper.SetDefault("server.port", 80)
	viper.SetDefault("server.realm", "Authentication Required")
	viper.SetDefault("server.cache.default-expire", "60s")
	viper.SetDefault("server.auth-ca", "/run/secrets/ca.pem")
	viper.BindEnv("server.auth-ca", "AUTH_CA_FILE")

	viper.SetDefault("server.skip-tls-verify", false)
	viper.BindEnv("server.skip-tls-verify", "SKIP_TLS_VERIFY")

	viper.SetDefault("server.ca-bundle", "/etc/ca-bundle.pem")
	viper.BindEnv("server.ca-bundle", "CA_BUNDLE_FILE")

	viper.SetDefault("auth.mincost", 15)

	// viper.SetDefault("server.audience", "")
}

func configInit() {
	viper.SetConfigName("auth-proxy")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./artifacts")
	viper.AddConfigPath("./test")
	viper.AddConfigPath("$HOME/.config")
	viper.AddConfigPath("$HOME/.auth-proxy")
	viper.AddConfigPath("/etc")
	viper.AddConfigPath("/etc/auth-proxy")
	viper.AddConfigPath("/usr/local/auth-proxy/etc")
	viper.AddConfigPath("/run/secrets")
	viper.AddConfigPath(".")

	configDefaults()

	viper.ReadInConfig()

	configFormatting()
}

func configFormatting() {
}

func zapConfig() zap.Config {
	var cfg zap.Config
	if viper.GetBool("debug") {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
		if !viper.GetBool("extra-logs") {
			cfg.Level.SetLevel(zapcore.WarnLevel)
		}
	}
	return cfg
}
