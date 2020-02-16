// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/app"
	"github.com/sqreen/go-agent/internal/backend"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
)

const (
	backoffRate          = 2
	backoffStartDuration = time.Second
	backoffMaxDuration   = time.Minute
)

type LoginError struct {
	err error
}

func NewLoginError(err error) LoginError {
	return LoginError{err}
}

func (e LoginError) Error() string {
	return fmt.Sprintf("could not login into sqreen: %s", e.err)
}

func (e LoginError) Cause() error {
	return e.err
}

func (e LoginError) Unwrap() error {
	return e.err
}

// Login to the backend. When the API request fails, retry for ever and after
// sleeping some time.
func appLogin(ctx context.Context, logger *plog.Logger, client *backend.Client, token string, appName string, appInfo *app.Info) (*api.AppLoginResponse, error) {
	if err := ValidateCredentialsConfiguration(token, appName); err != nil {
		return nil, NewLoginError(err)
	}

	_, bundleSignature, err := appInfo.Dependencies()
	if err != nil {
		logger.Error(withNotificationError{sqerrors.Wrap(err, "could not retrieve the program dependencies")})
	}

	appLoginReq := api.AppLoginRequest{
		VariousInfos:    *api.NewAppLoginRequest_VariousInfosFromFace((*appInfoAPIAdapter)(appInfo)),
		BundleSignature: bundleSignature,
		AgentType:       "golang",
		AgentVersion:    version,
		OsType:          app.GoBuildTarget(),
		Hostname:        appInfo.Hostname(),
		RuntimeVersion:  app.GoVersion(),
	}

	var appLoginRes *api.AppLoginResponse

	backoff := sqtime.NewBackoff(backoffStartDuration, backoffMaxDuration, backoffRate)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			appLoginRes, err = client.AppLogin(&appLoginReq, token, appName)
			if err == nil && appLoginRes.Status {
				return appLoginRes, nil
			}

			if appLoginRes != nil && !appLoginRes.Status {
				return nil, NewLoginError(errors.New(appLoginRes.Error))
			}

			logger.Infof("could not login into sqreen: %s", err)
			appLoginRes = nil
			d, max := backoff.Next()
			if max {
				return nil, NewLoginError(errors.New("login: maximum number of retries reached"))
			}
			logger.Debugf("login: retrying the request in %s", d)
			time.Sleep(d)
		}
	}
}

type InvalidCredentialsConfiguration struct {
	error
}

func (e InvalidCredentialsConfiguration) Unwrap() error {
	return e.error
}

func (e InvalidCredentialsConfiguration) Cause() error {
	return e.error
}

func ValidateCredentialsConfiguration(token, appName string) (err error) {
	defer func() {
		if err != nil {
			err = InvalidCredentialsConfiguration{err}
		}
	}()

	if token == "" {
		return sqerrors.New("missing token")
	}
	if strings.HasPrefix(token, config.BackendHTTPAPIOrganizationTokenPrefix) && appName == "" {
		return sqerrors.New("missing application name")
	}

	for _, r := range appName {
		if !unicode.IsPrint(r) {
			return sqerrors.Errorf("forbidden non-printable character `%q` in the application name `%q`", r, appName)
		}
	}

	for _, r := range token {
		if !unicode.IsPrint(r) {
			return sqerrors.Errorf("forbidden non-printable character `%q` in the token `%q`", r, token)
		}
	}

	return nil
}

// TrySendAppException is a special client function allowing to send app-level
// exceptions
func TrySendAppException(logger plog.DebugLogger, cfg *config.Config, exception error) {
	b := new(bytes.Buffer)
	payload := api.NewExceptionEventFromFace(NewExceptionEvent(exception, ""))
	err := json.NewEncoder(b).Encode(payload)
	if err != nil {
		return
	}
	endpoint := config.BackendHTTPAPIEndpoint.AppException
	req, err := http.NewRequest(endpoint.Method, cfg.BackendHTTPAPIBaseURL()+endpoint.URL, b)
	if err != nil {
		return
	}
	// TODO: factorize from here and Do() the request creation
	req.Header.Add(config.BackendHTTPAPIHeaderToken, cfg.BackendHTTPAPIToken())
	req.Header.Add(config.BackendHTTPAPIHeaderAppName, cfg.AppName())
	req.Header.Add("Content-Type", "application/json")

	logger.Debugf("sending app exception:\n%s\n", (*backend.HTTPRequestStringer)(req))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	logger.Debugf("received app exception response:\n%s\n", (*backend.HTTPResponseStringer)(res))
}
