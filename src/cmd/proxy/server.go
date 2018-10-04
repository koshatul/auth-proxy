package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/koshatul/auth-proxy/src/httpauth"
	"github.com/koshatul/auth-proxy/src/jwtauth"
	"github.com/koshatul/auth-proxy/src/legacy"
	"github.com/koshatul/auth-proxy/src/locator"
	"github.com/koshatul/auth-proxy/src/logformat"
	"github.com/koshatul/auth-proxy/src/proxy"
	"github.com/koshatul/jwt/src/jwt"
	cache "github.com/patrickmn/go-cache"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var cmdServer = &cobra.Command{
	Use:   "server",
	Short: "Start Server",
	Run:   serverCommand,
}

func init() {
	cmdServer.PersistentFlags().StringP("audience", "a", "tls-web-client-auth", "Server Audience")
	viper.BindPFlag("server.audience", cmdServer.PersistentFlags().Lookup("audience"))
	viper.BindEnv("server.audience", "AUDIENCE")

	cmdServer.PersistentFlags().StringP("backend", "u", "", "Backend URL (eg. 'http://docker-registry:5000/')")
	viper.BindPFlag("server.backend-uri", cmdServer.PersistentFlags().Lookup("backend"))
	viper.BindEnv("server.backend-uri", "BACKEND_URL")

	cmdServer.PersistentFlags().IntP("port", "p", 80, "HTTP Port")
	viper.BindPFlag("server.port", cmdServer.PersistentFlags().Lookup("port"))
	viper.BindEnv("server.port", "HTTP_PORT")

	cmdServer.PersistentFlags().Bool("pass-host-header", false, "Pass Host: header to backend server (default: false)")
	viper.BindPFlag("server.pass-host-header", cmdServer.PersistentFlags().Lookup("pass-host-header"))
	viper.BindEnv("server.pass-host-header", "PASS_HOST_HEADER")

	cmdServer.PersistentFlags().Bool("remove-auth", false, "Remove Authorization headers from HTTP proxy (default: false)")
	viper.BindPFlag("server.remove-authorization-header", cmdServer.PersistentFlags().Lookup("remove-auth"))
	viper.BindEnv("server.remove-authorization-header", "REMOVE_AUTH_HEADER")

	cmdServer.PersistentFlags().StringSliceP("legacy-user", "l", []string{}, "List of legacy users (username:password) that can authenticate, designed for allowing migration from a system with an old common login (allows it to work *temporarily*)")
	viper.BindPFlag("server.legacy-users", cmdServer.PersistentFlags().Lookup("legacy-user"))
	viper.BindEnv("server.legacy-users", "LEGACY_USERS")

	rootCmd.AddCommand(cmdServer)
}

func serverCommand(cmd *cobra.Command, args []string) {
	cfg := zapConfig()
	logger, _ := cfg.Build()
	defer logger.Sync()

	u, err := url.Parse(viper.GetString("server.backend-uri"))
	if err != nil {
		logger.Error("Parsing Backend URI", zap.String("URI", viper.GetString("server.backend-uri")), zap.Error(err))
		cmd.Help()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	authChan := make(chan *jwtauth.AuthRequest, 10)
	verifier, err := jwt.NewRSAVerifierFromFile(viper.GetString("server.audience"), viper.GetString("server.auth-ca"))
	if err != nil {
		logger.Error("Starting JWT Verifier", zap.Error(err))
		cmd.Help()
		cancel()
		return
	}
	go jwtauth.AuthRunner(ctx, logger, verifier, authChan)

	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	certs, err := ioutil.ReadFile(viper.GetString("server.ca-bundle"))
	if err == nil {
		logger.Debug("Appending custom certs", zap.String("ca-bundle", viper.GetString("server.ca-bundle")))
		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			logger.Debug("Failed to load custom certs", zap.String("ca-bundle", viper.GetString("server.ca-bundle")))
		}
	}

	authFunc := jwtauth.AuthCheckFunc(logger, authChan)

	cliLegacyUsers := viper.GetStringSlice("server.legacy-users")
	if len(cliLegacyUsers) > 0 {
		logger.Debug("loading legacy users")
		legacyUsers := make(map[string]legacy.AuthItem)
		for _, user := range cliLegacyUsers {
			s := strings.SplitN(user, ":", 2)
			if len(s) == 2 {
				logger.Debug("Appending user to legacyUsers", zap.String("username", s[0]))
				legacyUsers[s[0]] = legacy.AuthItem{
					Username: s[0],
					Password: s[1],
				}
			}
		}
		authFunc = legacy.AuthCheckFunc(logger, legacyUsers, authFunc)
	}

	honeycombLogger := log.New(os.Stdout, "", log.LstdFlags)
	proxyHandler := &proxy.Handler{
		Backend:        locator.NewEndpoint(u, viper.GetBool("server.pass-host-header")),
		HTTPProxy:      &proxy.HTTPProxy{},
		WebSocketProxy: &proxy.WebSocketProxy{},
		Logger:         honeycombLogger,
	}

	// tlsConfig := &tls.Config{
	// 	InsecureSkipVerify: viper.GetBool("server.skip-tls-verify"),
	// 	RootCAs:            rootCAs,
	// }
	// proxyHandler := handlers.ProxyHeaders(
	// 	proxy.NewSingleHostReverseProxy(u, viper.GetBool("server.pass-host-header"), tlsConfig),
	// )

	authenticator := &httpauth.BasicAuthHandler{
		Handler: proxyHandler,
		// Handler: handlers.ProxyHeaders(
		// 	oldproxy.NewSingleHostReverseProxy(u, viper.GetBool("server.pass-host-header"), tlsConfig),
		// ),
		RemoveAuth: viper.GetBool("server.remove-authorization-header"),
		BasicAuthWrapper: &httpauth.BasicAuthWrapper{
			Cache:    cache.New(viper.GetDuration("server.cache.default-expire"), time.Minute),
			Realm:    viper.GetString("server.realm"),
			AuthFunc: authFunc,
			Logger:   logger,
		},
	}

	s := http.NewServeMux()
	s.Handle("/", handlers.CustomLoggingHandler(
		os.Stdout,
		authenticator,
		logformat.WriteCombinedLog,
	))

	bindAddr := fmt.Sprintf("%s:%d", viper.GetString("server.address"), viper.GetInt("server.port"))

	logger.Info("Starting server", zap.String("audience", viper.GetString("server.audience")), zap.String("bind-addr", bindAddr), zap.String("proxy-uri", viper.GetString("server.backend-uri")))
	err = http.ListenAndServe(bindAddr, s)
	cancel()
	if err != nil {
		logger.Fatal("HTTP Server Error", zap.Error(err))
	}

}
