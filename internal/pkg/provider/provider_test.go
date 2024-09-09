package provider

import (
	"context"
	"math/rand"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/v-yarotksy/external-dns-opnsense-unbound-webhook-provider/internal/pkg/api"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

type fakeAPI struct {
	hostOverrides []api.HostOverride
	hostAliases   []api.HostAlias
}

func (f *fakeAPI) ListHostOverrides(_ context.Context) ([]api.HostOverride, error) {
	return f.hostOverrides, nil
}

func (f *fakeAPI) CreateHostOverride(_ context.Context, ho api.HostOverride) (api.HostOverride, error) {
	ho.ID = api.HostOverrideID(strconv.Itoa(rand.Int()))
	f.hostOverrides = append(f.hostOverrides, ho)
	return ho, nil
}

func (f *fakeAPI) DeleteHostOverride(_ context.Context, ho api.HostOverride) error {
	f.hostOverrides = slices.DeleteFunc(f.hostOverrides, func(e api.HostOverride) bool {
		return e == ho
	})
	return nil
}

func (f *fakeAPI) UpdateHostOverride(_ context.Context, ho api.HostOverride) error {
	for i, h := range f.hostOverrides {
		if ho.ID == h.ID {
			f.hostOverrides[i] = ho
		}
	}
	return nil
}

func (f *fakeAPI) ListHostAliases(_ context.Context, _ api.HostOverrideID) ([]api.HostAlias, error) {
	return f.hostAliases, nil
}

func (f *fakeAPI) CreateHostAlias(_ context.Context, ha api.HostAlias) (api.HostAlias, error) {
	ha.ID = api.HostAliasID(strconv.Itoa(rand.Int()))
	f.hostAliases = append(f.hostAliases, ha)
	return ha, nil
}

func (f *fakeAPI) UpdateHostAlias(_ context.Context, ha api.HostAlias) error {
	for i, h := range f.hostAliases {
		if ha.ID == h.ID {
			f.hostAliases[i] = ha
		}
	}
	return nil
}

func (f *fakeAPI) DeleteHostAlias(_ context.Context, ha api.HostAlias) error {
	f.hostAliases = slices.DeleteFunc(f.hostAliases, func(e api.HostAlias) bool {
		return e == ha
	})
	return nil
}

var _ api.API = &fakeAPI{}

func TestRecords(t *testing.T) {
	t.Run("returns an empty list when there are no records", func(t *testing.T) {
		fake := &fakeAPI{}
		provider := &unboundProvider{api: fake}

		res, err := provider.Records(context.Background())
		require.NoError(t, err)
		require.ElementsMatch(t, res, []*endpoint.Endpoint{})
	})

	t.Run("returns A records from Host Overrides and CNAME records from Host Aliases", func(t *testing.T) {
		fake := &fakeAPI{
			hostOverrides: []api.HostOverride{
				{
					ID:       api.HostOverrideID("berkin"),
					Hostname: "berkin",
					Domain:   "example.com",
					Server:   "127.0.0.1",
				},
			},
			hostAliases: []api.HostAlias{
				{
					ID:       api.HostAliasID("derkin"),
					Hostname: "derkin",
					Domain:   "example.com",
					Host:     "berkin.example.com",
					HostID:   api.HostOverrideID("berkin"),
				},
			},
		}
		provider := &unboundProvider{api: fake}

		res, err := provider.Records(context.Background())
		require.NoError(t, err)
		require.ElementsMatch(t, res, []*endpoint.Endpoint{
			{
				DNSName:    "berkin.example.com",
				RecordType: endpoint.RecordTypeA,
				Targets:    endpoint.NewTargets("127.0.0.1"),
			},
			{
				DNSName:    "derkin.example.com",
				RecordType: endpoint.RecordTypeCNAME,
				Targets:    endpoint.NewTargets("berkin.example.com"),
			},
		})
	})
}

func TestAdjustEndpoints(t *testing.T) {
	t.Run("removes anything but the first IP from A records", func(t *testing.T) {
		fake := &fakeAPI{}
		provider := &unboundProvider{api: fake}

		endpoints := []*endpoint.Endpoint{
			{
				DNSName:    "a.example.com",
				Targets:    endpoint.NewTargets("127.0.0.1", "127.0.0.2"),
				RecordType: endpoint.RecordTypeA,
			},
			{
				DNSName:    "cname.example.com",
				Targets:    endpoint.NewTargets("a.example.com"),
				RecordType: endpoint.RecordTypeCNAME,
			},
		}

		_, err := provider.AdjustEndpoints(endpoints)
		require.NoError(t, err)
		require.ElementsMatch(t, endpoints, []*endpoint.Endpoint{
			{
				DNSName:    "a.example.com",
				Targets:    endpoint.NewTargets("127.0.0.1"),
				RecordType: endpoint.RecordTypeA,
			},
			{
				DNSName:    "cname.example.com",
				Targets:    endpoint.NewTargets("a.example.com"),
				RecordType: endpoint.RecordTypeCNAME,
			},
		})
	})
}

func TestApplyChanges(t *testing.T) {
	t.Run("deletes Host Overrides when an A record is deleted", func(t *testing.T) {
		fake := &fakeAPI{
			hostOverrides: []api.HostOverride{
				{
					ID:       api.HostOverrideID("berkin"),
					Hostname: "berkin",
					Domain:   "example.com",
					Server:   "127.0.0.1",
				},
			},
		}
		provider := &unboundProvider{api: fake}

		err := provider.ApplyChanges(context.Background(), &plan.Changes{
			Delete: []*endpoint.Endpoint{
				{
					DNSName:    "berkin.example.com",
					Targets:    endpoint.NewTargets("127.0.0.1"),
					RecordType: endpoint.RecordTypeA,
				},
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, fake.hostOverrides, []api.HostOverride{})
	})

	t.Run("deletes Host Alias when a CNAME record is deleted", func(t *testing.T) {
		fake := &fakeAPI{
			hostOverrides: []api.HostOverride{
				{
					ID:       api.HostOverrideID("berkin"),
					Hostname: "berkin",
					Domain:   "example.com",
					Server:   "127.0.0.1",
				},
			},
			hostAliases: []api.HostAlias{
				{
					ID:       api.HostAliasID("derkin"),
					Hostname: "derkin",
					Domain:   "example.com",
					Host:     "berkin.example.com",
					HostID:   api.HostOverrideID("berkin"),
				},
			},
		}
		provider := &unboundProvider{api: fake}

		err := provider.ApplyChanges(context.Background(), &plan.Changes{
			Delete: []*endpoint.Endpoint{
				{
					DNSName:    "derkin.example.com",
					Targets:    endpoint.NewTargets("berkin.example.com"),
					RecordType: endpoint.RecordTypeCNAME,
				},
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, fake.hostAliases, []api.HostOverride{})
	})

	t.Run("creates a Host Override when an A record is created", func(t *testing.T) {
		fake := &fakeAPI{}
		provider := &unboundProvider{api: fake}

		err := provider.ApplyChanges(context.Background(), &plan.Changes{
			Create: []*endpoint.Endpoint{
				{
					DNSName:    "berkin.example.com",
					Targets:    endpoint.NewTargets("127.0.0.1"),
					RecordType: endpoint.RecordTypeA,
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, fake.hostOverrides, 1)
		require.Equal(t, "berkin", fake.hostOverrides[0].Hostname)
		require.Equal(t, "example.com", fake.hostOverrides[0].Domain)
		require.Equal(t, "127.0.0.1", fake.hostOverrides[0].Server)
		require.NotEmpty(t, fake.hostOverrides[0].ID)
	})

	t.Run("creates a Host Alias when a CNAME record is created", func(t *testing.T) {
		fake := &fakeAPI{
			hostOverrides: []api.HostOverride{
				{
					ID:       api.HostOverrideID("a"),
					Hostname: "a",
					Domain:   "example.com",
					Server:   "127.0.0.1",
				},
			},
		}
		provider := &unboundProvider{api: fake}

		err := provider.ApplyChanges(context.Background(), &plan.Changes{
			Create: []*endpoint.Endpoint{
				{
					DNSName:    "cname.example.com",
					Targets:    endpoint.NewTargets("a.example.com"),
					RecordType: endpoint.RecordTypeCNAME,
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, fake.hostAliases, 1)
		require.Equal(t, "cname", fake.hostAliases[0].Hostname)
		require.Equal(t, "example.com", fake.hostAliases[0].Domain)
		require.Equal(t, "a.example.com", fake.hostAliases[0].Host)
		require.Equal(t, api.HostOverrideID("a"), fake.hostAliases[0].HostID)
		require.NotEmpty(t, fake.hostAliases[0].ID)
	})

	t.Run("updates Host Overrides when an A record is updated", func(t *testing.T) {
		fake := &fakeAPI{
			hostOverrides: []api.HostOverride{
				{
					ID:       api.HostOverrideID("a"),
					Hostname: "a",
					Domain:   "example.com",
					Server:   "127.0.0.1",
				},
			},
		}
		provider := &unboundProvider{api: fake}

		err := provider.ApplyChanges(context.Background(), &plan.Changes{
			UpdateOld: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    endpoint.NewTargets("127.0.0.1"),
					RecordType: endpoint.RecordTypeA,
				},
			},
			UpdateNew: []*endpoint.Endpoint{
				{
					DNSName:    "a.example.com",
					Targets:    endpoint.NewTargets("127.0.0.2"),
					RecordType: endpoint.RecordTypeA,
				},
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, fake.hostOverrides, []api.HostOverride{
			{
				ID:       api.HostOverrideID("a"),
				Hostname: "a",
				Domain:   "example.com",
				Server:   "127.0.0.2",
			},
		})
	})

	t.Run("updates Host Alias when a CNAME record is updated", func(t *testing.T) {
		fake := &fakeAPI{
			hostOverrides: []api.HostOverride{
				{
					ID:       api.HostOverrideID("a"),
					Hostname: "a",
					Domain:   "example.com",
					Server:   "127.0.0.1",
				},
			},
			hostAliases: []api.HostAlias{
				{
					ID:       api.HostAliasID("cname"),
					Hostname: "cname",
					Domain:   "example.com",
					Host:     "a.example.com",
					HostID:   api.HostOverrideID("a"),
				},
			},
		}
		provider := &unboundProvider{api: fake}

		err := provider.ApplyChanges(context.Background(), &plan.Changes{
			UpdateOld: []*endpoint.Endpoint{
				{
					DNSName:    "cname.example.com",
					Targets:    endpoint.NewTargets("a.example.com"),
					RecordType: endpoint.RecordTypeCNAME,
				},
			},
			UpdateNew: []*endpoint.Endpoint{
				{
					DNSName:    "cname2.example.com",
					Targets:    endpoint.NewTargets("a.example.com"),
					RecordType: endpoint.RecordTypeCNAME,
				},
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, fake.hostAliases, []api.HostAlias{
			{
				ID:       api.HostAliasID("cname"),
				Hostname: "cname2",
				Domain:   "example.com",
				Host:     "a.example.com",
				HostID:   api.HostOverrideID("a"),
			},
		})
	})
}
