package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/koshatul/auth-proxy/httpauth"
	"github.com/koshatul/auth-proxy/jwtauth"
	"github.com/koshatul/auth-proxy/legacy"
	"github.com/koshatul/auth-proxy/logformat"
	"github.com/koshatul/auth-proxy/proxy"
	"github.com/koshatul/jwt/v2"
	"github.com/na4ma4/config"
	cache "github.com/patrickmn/go-cache"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// nolint: gochecknoglobals // cobra uses globals in main
var cmdServer = &cobra.Command{
	Use:   "server",
	Short: "Start Server",
	Run:   serverCommand,
}

// nolint:gochecknoinits // init is used in main for cobra
func init() {
	cmdServer.PersistentFlags().StringP("audience", "a", "tls-web-client-auth", "Server Audience")
	_ = viper.BindPFlag("server.audience", cmdServer.PersistentFlags().Lookup("audience"))
	_ = viper.BindEnv("server.audience", "AUDIENCE")

	cmdServer.PersistentFlags().StringP("backend", "u", "", "Backend URL (eg. 'http://docker-registry:5000/')")
	_ = viper.BindPFlag("server.backend-uri", cmdServer.PersistentFlags().Lookup("backend"))
	_ = viper.BindEnv("server.backend-uri", "BACKEND_URL")

	cmdServer.PersistentFlags().IntP("port", "p", 80, "HTTP Port")
	_ = viper.BindPFlag("server.port", cmdServer.PersistentFlags().Lookup("port"))
	_ = viper.BindEnv("server.port", "HTTP_PORT")

	cmdServer.PersistentFlags().Bool("pass-host-header", false, "Pass Host: header to backend server (default: false)")
	_ = viper.BindPFlag("server.pass-host-header", cmdServer.PersistentFlags().Lookup("pass-host-header"))
	_ = viper.BindEnv("server.pass-host-header", "PASS_HOST_HEADER")

	cmdServer.PersistentFlags().Bool("remove-auth", false, "Remove Authorization headers from HTTP proxy (default: false)")
	_ = viper.BindPFlag("server.remove-authorization-header", cmdServer.PersistentFlags().Lookup("remove-auth"))
	_ = viper.BindEnv("server.remove-authorization-header", "REMOVE_AUTH_HEADER")

	cmdServer.PersistentFlags().StringSliceP(
		"legacy-user",
		"l",
		[]string{},
		"List of legacy users (username:password) that can authenticate, designed "+
			"for allowing migration from a system with an old common login (allows it to work *temporarily*)",
	)

	_ = viper.BindPFlag("server.legacy-users", cmdServer.PersistentFlags().Lookup("legacy-user"))
	_ = viper.BindEnv("server.legacy-users", "LEGACY_USERS")

	rootCmd.AddCommand(cmdServer)
}

const userPassSepCount int = 2
const authChanSize int = 10

func addLegacyAuthFunc(
	logger *zap.Logger,
	cliLegacyUsers []string,
	authFunc httpauth.AuthProvider,
) httpauth.AuthProvider {
	if len(cliLegacyUsers) > 0 {
		logger.Debug("loading legacy users")

		legacyUsers := make(map[string]legacy.AuthItem)

		for _, user := range cliLegacyUsers {
			s := strings.SplitN(user, ":", userPassSepCount)

			if len(s) == userPassSepCount {
				logger.Debug("Appending user to legacyUsers", zap.String("username", s[0]))

				legacyUsers[s[0]] = legacy.AuthItem{
					Username: s[0],
					Password: s[1],
				}
			}
		}

		authFunc = legacy.AuthCheckFunc(logger, legacyUsers, authFunc)
	}

	return authFunc
}

func showHelp(cmd *cobra.Command) {
	_ = cmd.Help()
}

func verifierOrBust(cmd *cobra.Command, cfg config.Conf, logger *zap.Logger) (verifier jwt.Verifier) {
	var err error

	if verifier, err = jwt.NewRSAVerifierFromFile(
		cfg.GetString("server.audience"),
		cfg.GetString("server.auth-ca"),
	); err != nil {
		logger.Error("starting jwt verifier", zap.Error(err))
		showHelp(cmd)
		os.Exit(1)
	}

	return
}

func backendURIOrBust(cmd *cobra.Command, cfg config.Conf, logger *zap.Logger) (u *url.URL) {
	var err error

	if u, err = url.Parse(cfg.GetString("server.backend-uri")); err != nil {
		logger.Error("parsing backend URI", zap.String("URI", cfg.GetString("server.backend-uri")), zap.Error(err))
		showHelp(cmd)
		os.Exit(1)
	}

	return
}

func buildCertPool(cfg config.Conf, logger *zap.Logger) *x509.CertPool {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	certs, err := ioutil.ReadFile(cfg.GetString("server.ca-bundle"))
	if err == nil {
		logger.Debug("appending custom certs", zap.String("ca-bundle", cfg.GetString("server.ca-bundle")))

		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			logger.Debug("failed to load custom certs", zap.String("ca-bundle", cfg.GetString("server.ca-bundle")))
		}
	}

	return rootCAs
}

func serverCommand(cmd *cobra.Command, args []string) {
	cfg := config.NewViperConfigFromViper(viper.GetViper(), "auth-proxy")
	authChan := make(chan *jwtauth.AuthRequest, authChanSize)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := cfg.ZapConfig().Build()
	defer logger.Sync() //nolint:errcheck

	u := backendURIOrBust(cmd, cfg, logger)
	verifier := verifierOrBust(cmd, cfg, logger)

	go jwtauth.AuthRunner(ctx, logger, verifier, authChan)

	rootCAs := buildCertPool(cfg, logger)

	//nolint:gosec // defaults to false, but it's up to the user
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.GetBool("server.skip-tls-verify"),
		RootCAs:            rootCAs,
	}

	authFunc := jwtauth.AuthCheckFunc(logger, authChan)
	cliLegacyUsers := cfg.GetStringSlice("server.legacy-users")
	authFunc = addLegacyAuthFunc(logger, cliLegacyUsers, authFunc)
	s := http.NewServeMux()
	authenticator := &httpauth.BasicAuthHandler{
		Handler: handlers.ProxyHeaders(
			proxy.NewSingleHostReverseProxy(u, cfg.GetBool("server.pass-host-header"), tlsConfig),
		),
		RemoveAuth: cfg.GetBool("server.remove-authorization-header"),
		BasicAuthWrapper: &httpauth.BasicAuthWrapper{
			Cache:         cache.New(cfg.GetDuration("server.cache.default-expire"), time.Minute),
			Realm:         cfg.GetString("server.realm"),
			AuthFunc:      authFunc,
			Logger:        logger,
			CacheDuration: cfg.GetDuration("server.cache.default-expire"),
		},
	}

	s.Handle("/", handlers.CustomLoggingHandler(
		os.Stdout,
		authenticator,
		logformat.WriteCombinedLog,
	))

	bindAddr := fmt.Sprintf("%s:%d", cfg.GetString("server.address"), cfg.GetInt("server.port"))

	logger.Info("starting server",
		zap.String("audience", cfg.GetString("server.audience")),
		zap.String("bind-addr", bindAddr),
		zap.String("proxy-uri", cfg.GetString("server.backend-uri")),
	)

	if err := http.ListenAndServe(bindAddr, s); err != nil {
		logger.Fatal("HTTP Server Error", zap.Error(err))
	}
}
