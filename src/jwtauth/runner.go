package jwtauth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/koshatul/auth-proxy/src/httpauth"
	"github.com/koshatul/jwt/src/jwt"
	"go.uber.org/zap"
)

// AuthRequest is the authentication request and a return channel for the response.
type AuthRequest struct {
	Token         []byte
	ReturnChannel chan *AuthResponse
}

// AuthResponse is contains the response for a `AuthRequest`
type AuthResponse struct {
	Result jwt.VerifyResult
	Error  error
}

// AuthRunner is the routine that runs the authentication checking channels
func AuthRunner(ctx context.Context, logger *zap.Logger, verifier jwt.Verifier, authChan chan *AuthRequest) {
	for {
		request := <-authChan
		go doAuthRunner(ctx, logger, verifier, request)
	}
}

// doAuthRunner is the actual authentication check process (separated so it can be tested and defers will work)
func doAuthRunner(ctx context.Context, logger *zap.Logger, verifier jwt.Verifier, request *AuthRequest) {
	result, err := verifier.Verify(request.Token)
	if err != nil {
		logger.Debug("Error Verifying Token", zap.Error(err))
		request.ReturnChannel <- &AuthResponse{
			Error: err,
		}
		return
	}
	if strings.EqualFold(result.Subject, "") {
		logger.Debug("Verifying Token", zap.Error(errors.New("username is empty")))
		request.ReturnChannel <- &AuthResponse{
			Error: err,
		}
		return
	}

	if result.IsOnline {
		request.ReturnChannel <- &AuthResponse{
			Result: result,
			Error:  errors.New("online tokens can not be validated"),
		}
	} else {
		request.ReturnChannel <- &AuthResponse{
			Result: result,
			Error:  nil,
		}
	}
}

// AuthCheckFunc returns a authentication check function for use with `httpauth.BasicAuth()``
func AuthCheckFunc(logger *zap.Logger, authChan chan *AuthRequest) httpauth.AuthProvider {
	return func(username, password string, r *http.Request) (string, bool) {
		recCh := make(chan *AuthResponse)
		authChan <- &AuthRequest{
			Token:         []byte(username),
			ReturnChannel: recCh,
		}
		response := <-recCh

		if response.Error == nil && !strings.EqualFold(response.Result.Subject, "") {
			logger.Debug("Auth Success", zap.String("username", response.Result.Subject), zap.Bool("online", response.Result.IsOnline), zap.String("uuid", response.Result.ID), zap.Error(response.Error))
			return response.Result.Subject, true
		}
		logger.Info("Auth Failure", zap.String("username", response.Result.Subject), zap.Bool("online", response.Result.IsOnline), zap.String("uuid", response.Result.ID), zap.Error(response.Error))
		return "", false
	}
}
