package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/v-yarotksy/external-dns-opnsense-unbound-webhook-provider/internal/pkg/api"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

type Option func(*unboundProvider)

// OPNSense runs with self-signed cert
func WithInsecureClient() Option {
	return func(p *unboundProvider) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		p.client.Transport = tr
	}
}

func WithDomainFilter(domains []string) Option {
	return func(p *unboundProvider) {
		p.domains = append(p.domains, domains...)
	}
}

func NewUnboundProvider(baseURL, apiKey, apiSecret string, opts ...Option) (*unboundProvider, error) {
	client := http.DefaultClient

	api, err := api.NewUnboundClient(baseURL, apiKey, apiSecret, client)
	if err != nil {
		return nil, fmt.Errorf("failed to make unbound API client: %w", err)
	}

	provider := &unboundProvider{api: api, client: client}

	for _, opt := range opts {
		opt(provider)
	}

	return provider, nil
}

type unboundProvider struct {
	api     api.API
	client  *http.Client
	domains []string
}

func (p *unboundProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	res, err := p.api.ListHostOverrides(ctx)
	if err != nil {
		slog.Error("failed to list A records", slog.Any("error", err))
		return nil, err
	}
	result := make([]*endpoint.Endpoint, 0, len(res))
	for _, r := range res {
		result = append(result, r.Endpoint())

		cnameRes, err := p.api.ListHostAliases(ctx, r.ID)
		if err != nil {
			slog.Error("failed to list CNAME records", slog.Any("hostOverride", r), slog.Any("error", err))
			return nil, err
		}

		for _, cr := range cnameRes {
			result = append(result, cr.Endpoint())
		}
	}

	slog.Info("list records", slog.Any("result", result))

	return result, nil
}

func (p *unboundProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	if !changes.HasChanges() {
		slog.Debug("No changes")
		return nil
	}

	hostOverrides, err := p.api.ListHostOverrides(ctx)
	if err != nil {
		slog.Error("failed to list A records", slog.Any("error", err))
		return fmt.Errorf("failed to list A records: %w", err)
	}

	aRecordsByDNSName := make(map[string]api.HostOverride, len(hostOverrides))
	for _, ho := range hostOverrides {
		aRecordsByDNSName[ho.DNSName()] = ho
	}

	cnameRecordsByDNSName := make(map[string]api.HostAlias, 100)
	for _, ho := range hostOverrides {
		res, err := p.api.ListHostAliases(ctx, ho.ID)
		if err != nil {
			slog.Error("failed to list CNAME records", slog.Any("hostOverride", ho), slog.Any("error", err))
			return err
		}
		for _, ha := range res {
			cnameRecordsByDNSName[ha.DNSName()] = ha
		}
	}

	for _, ep := range changes.Delete {
		logger := slog.With(slog.String("op", "delete"), slog.Any("endpoint", ep))

		switch ep.RecordType {
		case endpoint.RecordTypeA:
			if ho, ok := aRecordsByDNSName[ep.DNSName]; ok {
				if err := p.api.DeleteHostOverride(ctx, ho); err != nil {
					logger.Error("failed to delete host override", slog.Any("hostOverride", ho))
					return fmt.Errorf("failed to delete host override: %w", err)
				} else {
					logger.Info("deleted Host Override", slog.Any("hostOverride", ho))
					delete(aRecordsByDNSName, ep.DNSName)
				}

			} else {
				logger.Warn("Host Override not found")
			}
		case endpoint.RecordTypeCNAME:
			if ha, ok := cnameRecordsByDNSName[ep.DNSName]; ok {
				if err := p.api.DeleteHostAlias(ctx, ha); err != nil {
					logger.Error("failed to delete host alias", slog.Any("hostAlias", ha))
					return fmt.Errorf("failed to delete host alias: %w", err)
				} else {
					logger.Info("deleted Host Alias", slog.Any("hostAlias", ha))
					delete(cnameRecordsByDNSName, ep.DNSName)
				}

			} else {
				logger.Warn("Host Alias not found")
			}
		default:
			logger.Warn("unsupported record type")
		}
	}

	for _, ep := range changes.Create {
		logger := slog.With(slog.String("op", "create"), slog.Any("endpoint", ep))

		var err error

		switch ep.RecordType {
		case endpoint.RecordTypeA:
			ho := api.HostOverride{}
			ho.Update(ep)
			if ho, err = p.api.CreateHostOverride(ctx, ho); err != nil {
				logger.Error("failed to create host override", slog.Any("hostOverride", ho))
				return fmt.Errorf("failed to create host override: %w", err)
			} else {
				logger.Info("created Host Override", slog.Any("hostOverride", ho))
				aRecordsByDNSName[ho.DNSName()] = ho
			}
		case endpoint.RecordTypeCNAME:
			if ho, ok := aRecordsByDNSName[ep.Targets[0]]; ok {
				ha := api.HostAlias{HostID: ho.ID}
				ha.Update(ep)
				if ha, err = p.api.CreateHostAlias(ctx, ha); err != nil {
					logger.Error("failed to create host alias", slog.Any("hostAlias", ha), slog.Any("hostOverride", ho))
					return fmt.Errorf("failed to create host alias: %w", err)
				} else {
					logger.Info("created Host Alias", slog.Any("hostAlias", ha), slog.Any("hostOverride", ho))
					cnameRecordsByDNSName[ha.DNSName()] = ha
				}
			} else {
				logger.Warn("Target Host Override not found for Host Alias")
				return fmt.Errorf("failed to create host alias: target host override not found")
			}
		default:
			logger.Warn("unsupported record type")
		}
	}

	// Record type changes are handled for us via delete/create
	for i, oldEP := range changes.UpdateOld {
		newEP := changes.UpdateNew[i]

		logger := slog.With(slog.String("op", "update"), slog.Any("oldEndpoint", oldEP), slog.Any("newEndpoint", newEP))

		switch oldEP.RecordType {
		case endpoint.RecordTypeA:
			if ho, ok := aRecordsByDNSName[oldEP.DNSName]; ok {
				ho.Update(newEP)
				if err := p.api.UpdateHostOverride(ctx, ho); err != nil {
					logger.Error("failed to update host override", slog.Any("hostOverride", ho))
					return fmt.Errorf("failed to update host override: %w", err)
				} else {
					logger.Info("updated Host Override", slog.Any("hostOverride", ho))
					aRecordsByDNSName[ho.DNSName()] = ho
				}
			} else {
				logger.Warn("Host Override not found")
			}
		case endpoint.RecordTypeCNAME:
			if haOld, ok := cnameRecordsByDNSName[oldEP.DNSName]; ok {
				if ho, ok := aRecordsByDNSName[newEP.Targets[0]]; ok {
					ha := haOld
					ha.Update(newEP)
					ha.HostID = ho.ID
					if err := p.api.UpdateHostAlias(ctx, ha); err != nil {
						logger.Error("failed to update host alias", slog.Any("hostAlias", ha), slog.Any("hostOverride", ho))
						return fmt.Errorf("failed to update host alias: %w", err)
					} else {
						logger.Info("updated Host Alias", slog.Any("hostAlias", ha), slog.Any("hostOverride", ho))
						cnameRecordsByDNSName[ha.DNSName()] = ha
					}
				} else {
					logger.Warn("Target Host Override not found for Host Alias")
					return fmt.Errorf("failed to update host alias: target host override not found")
				}
			} else {
				logger.Warn("Host Alias not found")
				return fmt.Errorf("host alias not found")
			}
		default:
			logger.Warn("unsupported record type")
		}
	}

	return nil
}

func (u *unboundProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	for _, e := range endpoints {
		if e.RecordType == endpoint.RecordTypeA {
			// Unbound only supports one IP address per A record
			e.Targets = endpoint.NewTargets(e.Targets[0])
		}
	}
	return endpoints, nil
}

func (u *unboundProvider) GetDomainFilter() endpoint.DomainFilter {
	return endpoint.DomainFilter{
		Filters: u.domains,
	}
}

var _ provider.Provider = &unboundProvider{}
