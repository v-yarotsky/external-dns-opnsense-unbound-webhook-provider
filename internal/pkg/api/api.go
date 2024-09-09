package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"sigs.k8s.io/external-dns/endpoint"
)

type API interface {
	ListHostOverrides(context.Context) ([]HostOverride, error)
	CreateHostOverride(context.Context, HostOverride) (HostOverride, error)
	DeleteHostOverride(context.Context, HostOverride) error
	UpdateHostOverride(context.Context, HostOverride) error
	ListHostAliases(context.Context, HostOverrideID) ([]HostAlias, error)
	CreateHostAlias(context.Context, HostAlias) (HostAlias, error)
	UpdateHostAlias(context.Context, HostAlias) error
	DeleteHostAlias(context.Context, HostAlias) error
}

type unboundClient struct {
	URL       *url.URL
	APIKey    string
	APISecret string

	client *http.Client
}

func NewUnboundClient(baseURL string, apiKey, apiSecret string, client *http.Client) (*unboundClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("bad base url %q: %w", baseURL, err)
	}

	return &unboundClient{
		URL:       u,
		APIKey:    apiKey,
		APISecret: apiSecret,
		client:    client,
	}, nil
}

type HostOverrideID string

type HostOverride struct {
	ID       HostOverrideID
	Hostname string
	Domain   string
	Server   string
}

func (r *HostOverride) Endpoint() *endpoint.Endpoint {
	return &endpoint.Endpoint{
		DNSName:    fmt.Sprintf("%s.%s", r.Hostname, r.Domain),
		Targets:    endpoint.NewTargets(r.Server),
		RecordType: "A",
	}
}

func (r *HostOverride) Update(ep *endpoint.Endpoint) {
	parts := strings.SplitN(ep.DNSName, ".", 2)
	r.Hostname = parts[0]
	r.Domain = parts[1]
	r.Server = ep.Targets[0]
}

func (r *HostOverride) DNSName() string {
	return fmt.Sprintf("%s.%s", r.Hostname, r.Domain)
}

type HostAliasID string

type HostAlias struct {
	ID          HostAliasID    `json:"uuid"`        // "f61b5bdb-8b51-46ff-a47f-ace0f5ca94b7"
	Enabled     string         `json:"enabled"`     // "1"
	Host        string         `json:"host"`        // "traefik.home.yarotsky.me"
	HostID      HostOverrideID `json:"-"`           // "2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c"
	Hostname    string         `json:"hostname"`    // "test"
	Domain      string         `json:"domain"`      // "home.yarotsky.me"
	Description string         `json:"description"` // ""
}

func (r *HostAlias) Endpoint() *endpoint.Endpoint {
	return &endpoint.Endpoint{
		DNSName:    fmt.Sprintf("%s.%s", r.Hostname, r.Domain),
		Targets:    endpoint.NewTargets(r.Host),
		RecordType: "CNAME",
	}
}

func (r *HostAlias) Update(ep *endpoint.Endpoint) {
	parts := strings.SplitN(ep.DNSName, ".", 2)
	r.Hostname = parts[0]
	r.Domain = parts[1]
	r.Host = ep.Targets[0]
}

func (r *HostAlias) DNSName() string {
	return fmt.Sprintf("%s.%s", r.Hostname, r.Domain)
}

type HostOverrideRequest struct {
	Host HostOverrideRequestHost `json:"host"`
}

type HostOverrideRequestHost struct {
	Enabled     string `json:"enabled"`     // "1"
	Hostname    string `json:"hostname"`    // "ha"
	Domain      string `json:"domain"`      // "home.yarotsky.me"
	RR          string `json:"rr"`          // "A"
	MXPrio      string `json:"mxprio"`      // ""
	MX          string `json:"mx"`          // ""
	Server      string `json:"server"`      // "192.168.1.13"
	Description string `json:"description"` // ""
}

type AddHostOverrideResponse struct {
	Result      string                 `json:"result"` // "saved"
	ID          HostOverrideID         `json:"uuid"`   // "2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c"
	Validations map[string]interface{} `json:"validations,omitempty"`
}

type UpdateHostOverrideResponse struct {
	Result      string                 `json:"result"` // "saved"
	Validations map[string]interface{} `json:"validations,omitempty"`
}

type DeleteHostOverrideResponse struct {
	Result string `json:"result"` // "deleted"
}

type SearchHostOverrideRequest struct {
	Current  int `json:"current"`
	RowCount int `json:"rowCount"`
}

type SearchHostOverrideResponse struct {
	Rows     []SearchHostOverride `json:"rows"`
	RowCount int                  `json:"rowCount"`
	Total    int                  `json:"total"`
	Current  int                  `json:"current"`
}

type SearchHostOverride struct {
	ID          HostOverrideID `json:"uuid"`        // "2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c"
	Enabled     string         `json:"enabled"`     // "1"
	Hostname    string         `json:"hostname"`    // "ha"
	Domain      string         `json:"domain"`      // "home.yarotsky.me"
	Server      string         `json:"server"`      // "192.168.1.13"
	Description string         `json:"description"` // ""
}

type SetHostOverrideResponse struct {
	Result      string                 `json:"result"` // "saved"
	Validations map[string]interface{} `json:"validations,omitempty"`
}

type SearchHostAliasRequest struct {
	Current  int            `json:"current"`
	RowCount int            `json:"rowCount"`
	HostID   HostOverrideID `json:"host"`
}

type SearchHostAliasResponse struct {
	Rows     []SearchHostAlias `json:"rows"`
	RowCount int               `json:"rowCount"`
	Total    int               `json:"total"`
	Current  int               `json:"current"`
}

type SearchHostAlias struct {
	ID          HostAliasID `json:"uuid"`        // "18b07c57-fce4-43ad-8bd8-5fb0e8777800"
	Enabled     string      `json:"enabled"`     // "1"
	Hostname    string      `json:"hostname"`    // "ha"
	Domain      string      `json:"domain"`      // "home.yarotsky.me"
	Host        string      `json:"host"`        // "traefik.home.yarotsky.me"
	Description string      `json:"description"` // ""
}

type HostAliasRequest struct {
	Alias HostAliasRequestAlias `json:"alias"`
}

type HostAliasRequestAlias struct {
	Enabled     string         `json:"enabled"`     // "1"
	HostID      HostOverrideID `json:"host"`        // "2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c"
	Hostname    string         `json:"hostname"`    // "ha"
	Domain      string         `json:"domain"`      // "home.yarotsky.me"
	Description string         `json:"description"` // ""
}

type AddHostAliasResponse struct {
	Result      string                 `json:"result"` // "saved"
	ID          HostAliasID            `json:"uuid"`   // "2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c"
	Validations map[string]interface{} `json:"validations,omitempty"`
}

type UpdateHostAliasResponse struct {
	Result      string                 `json:"result"` // "saved"
	Validations map[string]interface{} `json:"validations,omitempty"`
}

type DeleteHostAliasResponse struct {
	Result string `json:"result"` // "deleted"
}

func (u *unboundClient) ListHostOverrides(ctx context.Context) ([]HostOverride, error) {
	req := &SearchHostOverrideRequest{Current: 1, RowCount: -1}

	var res SearchHostOverrideResponse

	if err := u.postJSON(ctx, "/api/unbound/settings/searchHostOverride/", req, &res); err != nil {
		return nil, err
	}

	result := make([]HostOverride, 0, len(res.Rows))

	for _, row := range res.Rows {
		rec := HostOverride{
			ID:       HostOverrideID(row.ID),
			Hostname: row.Hostname,
			Domain:   row.Domain,
			Server:   row.Server,
		}
		result = append(result, rec)
	}

	return result, nil
}

func (u *unboundClient) CreateHostOverride(ctx context.Context, rec HostOverride) (HostOverride, error) {
	req := &HostOverrideRequest{
		Host: HostOverrideRequestHost{
			Enabled:  "1",
			Hostname: rec.Hostname,
			Domain:   rec.Domain,
			RR:       "A",
			Server:   rec.Server,
		},
	}

	var res AddHostOverrideResponse

	if err := u.postJSON(ctx, "/api/unbound/settings/addHostOverride/", req, &res); err != nil {
		return rec, err
	}

	if res.Result != "saved" {
		slog.Error("addHostOverride failed", slog.Any("hostOverride", rec), slog.Any("response", res))
		return rec, fmt.Errorf("addHostOverride failed: %s", res.Result)
	}

	rec.ID = res.ID

	return rec, nil
}

func (u *unboundClient) DeleteHostOverride(ctx context.Context, rec HostOverride) error {
	var res DeleteHostOverrideResponse

	if err := u.postJSON(ctx, "/api/unbound/settings/delHostOverride/"+string(rec.ID), map[string]interface{}{}, &res); err != nil {
		return err
	}

	if res.Result != "deleted" {
		slog.Error("delHostOverride failed", slog.Any("hostOverride", rec), slog.Any("response", res))
		return fmt.Errorf("delHostOverride failed: %s", res.Result)
	}

	return nil
}

func (u *unboundClient) UpdateHostOverride(ctx context.Context, rec HostOverride) error {
	var res UpdateHostOverrideResponse

	req := &HostOverrideRequest{
		Host: HostOverrideRequestHost{
			Enabled:  "1",
			Hostname: rec.Hostname,
			Domain:   rec.Domain,
			RR:       "A",
			Server:   rec.Server,
		},
	}

	if err := u.postJSON(ctx, "/api/unbound/settings/setHostOverride/"+string(rec.ID), req, &res); err != nil {
		return err
	}

	if res.Result != "saved" {
		slog.Error("setHostOverride failed", slog.Any("hostOverride", rec), slog.Any("response", res))
		return fmt.Errorf("setHostOverride failed: %s", res.Result)
	}

	return nil
}

func (u *unboundClient) ListHostAliases(ctx context.Context, id HostOverrideID) ([]HostAlias, error) {
	req := &SearchHostAliasRequest{
		Current:  1,
		RowCount: -1,
		HostID:   id,
	}

	var res SearchHostAliasResponse

	if err := u.postJSON(ctx, "/api/unbound/settings/searchHostAlias/", req, &res); err != nil {
		return nil, err
	}

	result := make([]HostAlias, 0, len(res.Rows))
	for _, row := range res.Rows {
		rec := HostAlias{
			ID:       HostAliasID(row.ID),
			Hostname: row.Hostname,
			Domain:   row.Domain,
			Host:     row.Host,
			HostID:   id,
		}
		result = append(result, rec)
	}

	return result, nil
}

func (u *unboundClient) CreateHostAlias(ctx context.Context, rec HostAlias) (HostAlias, error) {
	req := &HostAliasRequest{
		Alias: HostAliasRequestAlias{
			Enabled:  "1",
			Hostname: rec.Hostname,
			Domain:   rec.Domain,
			HostID:   rec.HostID,
		},
	}

	var res AddHostAliasResponse

	if err := u.postJSON(ctx, "/api/unbound/settings/addHostAlias/", req, &res); err != nil {
		return rec, err
	}

	if res.Result != "saved" {
		slog.Error("addHostAlias failed", slog.Any("alias", rec), slog.Any("response", res))
		return rec, fmt.Errorf("addHostAlias failed: %s", res.Result)
	}

	rec.ID = res.ID

	return rec, nil
}

func (u *unboundClient) UpdateHostAlias(ctx context.Context, rec HostAlias) error {
	req := &HostAliasRequest{
		Alias: HostAliasRequestAlias{
			Enabled:  "1",
			Hostname: rec.Hostname,
			Domain:   rec.Domain,
			HostID:   rec.HostID,
		},
	}

	var res UpdateHostAliasResponse

	if err := u.postJSON(ctx, "/api/unbound/settings/setHostAlias/"+string(rec.ID), req, &res); err != nil {
		return err
	}

	if res.Result != "saved" {
		slog.Error("setHostAlias failed", slog.Any("alias", rec), slog.Any("response", res))
		return fmt.Errorf("setHostAlias failed: %s", res.Result)
	}

	return nil
}

// DelHostAlias deletes a CNAME record.
// rec MUST have ID set.
func (u *unboundClient) DeleteHostAlias(ctx context.Context, rec HostAlias) error {
	var res DeleteHostAliasResponse

	if err := u.postJSON(ctx, "/api/unbound/settings/delHostAlias/"+string(rec.ID), map[string]interface{}{}, &res); err != nil {
		return err
	}

	if res.Result != "deleted" {
		slog.Error("delHostAlias failed", slog.Any("alias", rec), slog.Any("response", res))
		return fmt.Errorf("delHostAlias failed: %s", res.Result)
	}

	return nil
}

func (u *unboundClient) postJSON(ctx context.Context, path string, body interface{}, out interface{}) error {
	logger := slog.With(slog.String("path", path), slog.Any("body", body))

	reqBodyJSON, err := json.Marshal(body)
	if err != nil {
		logger.Error("failed to serialize request body", slog.Any("error", err))
		return fmt.Errorf("failed to serialize request body: %w", err)
	}

	url := u.URL.JoinPath(path)
	req, err := http.NewRequestWithContext(ctx, "POST", url.String(), bytes.NewReader(reqBodyJSON))
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	req.SetBasicAuth(u.APIKey, u.APISecret)

	if err != nil {
		logger.Error("failed to prepare request", slog.Any("error", err))
		return fmt.Errorf("failed to prepare request: %w", err)
	}

	res, err := u.client.Do(req)
	if err != nil {
		logger.Error("request failed", slog.Any("error", err))
		return fmt.Errorf("request failed: %w", err)
	}

	err = json.NewDecoder(res.Body).Decode(out)
	if err != nil {
		logger.Error("failed to deserialize response", slog.Any("error", err))
		return fmt.Errorf("failed to deserialize response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		logger.Error("request failed", slog.Any("status", res.StatusCode))
		return fmt.Errorf("request failed: %d", res.StatusCode)
	}

	return nil
}

var _ API = &unboundClient{}
