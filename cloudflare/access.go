package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AccessService provides Cloudflare Access and Zero Trust API operations.
type AccessService struct {
	client *Client
}

// Access returns the Access service API.
func (c *Client) Access() *AccessService {
	return &AccessService{client: c}
}

// Do performs a scoped Access API request.
func (a *AccessService) Do(
	ctx context.Context,
	scope Scope,
	method string,
	endpoint string,
	params url.Values,
	requestBody any,
	out any,
	reqOpts ...RequestOption,
) error {
	prefix, err := scope.PathPrefix()
	if err != nil {
		return err
	}

	cleanEndpoint := strings.TrimPrefix(strings.TrimSpace(endpoint), "/")
	if cleanEndpoint == "" {
		return errors.New("access endpoint must not be empty")
	}

	return a.client.DoWithOptions(
		ctx,
		method,
		fmt.Sprintf("/%s/%s", prefix, cleanEndpoint),
		params,
		requestBody,
		out,
		reqOpts...,
	)
}

// CreateIdentityProvider creates an Access identity provider (login method).
func (a *AccessService) CreateIdentityProvider(
	ctx context.Context,
	accountID string,
	requestBody any,
	out any,
	reqOpts ...RequestOption,
) error {
	return a.Do(
		ctx,
		AccountScope(accountID),
		http.MethodPost,
		"/access/identity_providers",
		nil,
		requestBody,
		out,
		reqOpts...,
	)
}

// CreateApplication creates an Access application at account or zone scope.
func (a *AccessService) CreateApplication(
	ctx context.Context,
	scope Scope,
	requestBody any,
	out any,
	reqOpts ...RequestOption,
) error {
	return a.Do(
		ctx,
		scope,
		http.MethodPost,
		"/access/apps",
		nil,
		requestBody,
		out,
		reqOpts...,
	)
}

// CreateReusablePolicy creates a reusable Access policy at account scope.
func (a *AccessService) CreateReusablePolicy(
	ctx context.Context,
	accountID string,
	requestBody any,
	out any,
	reqOpts ...RequestOption,
) error {
	return a.Do(
		ctx,
		AccountScope(accountID),
		http.MethodPost,
		"/access/policies",
		nil,
		requestBody,
		out,
		reqOpts...,
	)
}

// CreateApplicationPolicy creates an application-scoped Access policy.
func (a *AccessService) CreateApplicationPolicy(
	ctx context.Context,
	scope Scope,
	appID string,
	requestBody any,
	out any,
	reqOpts ...RequestOption,
) error {
	cleanAppID := strings.TrimSpace(appID)
	if cleanAppID == "" {
		return errors.New("app ID must not be empty")
	}

	return a.Do(
		ctx,
		scope,
		http.MethodPost,
		fmt.Sprintf("/access/apps/%s/policies", url.PathEscape(cleanAppID)),
		nil,
		requestBody,
		out,
		reqOpts...,
	)
}
