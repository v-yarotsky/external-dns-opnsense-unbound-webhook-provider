package main

import (
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/v-yarotksy/external-dns-opnsense-unbound-webhook-provider/internal/pkg/provider"
	"sigs.k8s.io/external-dns/provider/webhook/api"
)

type stringSliceFlag []string

func (i *stringSliceFlag) String() string {
	return strings.Join(*i, ",")
}

func (i *stringSliceFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var baseURL, apiKey, apiSecret string
	var domains stringSliceFlag

	flag.StringVar(&baseURL, "base-url", "https://192.168.1.1", "OPNSense API base URL")
	flag.StringVar(&apiKey, "api-key", "", "OPNSense API key")
	flag.StringVar(&apiSecret, "api-secret", "", "OPNSense API secret")
	flag.Var(&domains, "domains", "Domain filter. Can be used multiple times. "+
		"foo.com means foo.com and anything that ends in .foo.com")

	if baseURL == "" {
		baseURL = os.Getenv("UNBOUND_BASE_URL")
	}

	if apiKey == "" {
		apiKey = os.Getenv("UNBOUND_API_KEY")
	}

	if apiSecret == "" {
		apiSecret = os.Getenv("UNBOUND_API_SECRET")
	}

	if len(domains) == 0 {
		domains = strings.Split(os.Getenv("UNBOUND_DOMAIN_FILTER"), ",")
	}

	if baseURL == "" {
		slog.Error("-base-url or UNBOUND_BASE_URL is required")
		os.Exit(1)
	}

	if apiKey == "" {
		slog.Error("-api-key or UNBOUND_API_KEY is required")
		os.Exit(1)
	}

	if apiSecret == "" {
		slog.Error("-api-secret or UNBOUND_API_SECRET is required")
		os.Exit(1)
	}

	prov, err := provider.NewUnboundProvider(
		baseURL,
		apiKey,
		apiSecret,
		provider.WithInsecureClient(),
		provider.WithDomainFilter(domains),
	)
	if err != nil {
		slog.Error("failed to create Unbound provider", slog.Any("error", err))
		os.Exit(1)
	}

	api.StartHTTPApi(prov, nil, 5*time.Second, 5*time.Second, ":8888")
}
