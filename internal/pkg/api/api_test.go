package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/v-yarotksy/external-dns-opnsense-unbound-webhook-provider/internal/pkg/api"
)

var (
	mux    *http.ServeMux
	server *httptest.Server
	client api.API
)

func setup(t *testing.T) (api.API, func()) {
	t.Helper()

	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	client, _ = api.NewUnboundClient(server.URL, "fakeapikey", "fakeapisecret", http.DefaultClient)

	return client, func() {
		server.Close()
	}
}

func fixture(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile("testdata/fixtures/" + path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestListHostOverrides(t *testing.T) {
	t.Run("returns host overrides", func(t *testing.T) {
		client, teardown := setup(t)
		t.Cleanup(teardown)

		mux.HandleFunc("/api/unbound/settings/searchHostOverride/", func(w http.ResponseWriter, r *http.Request) {
			var req api.SearchHostOverrideRequest
			json.NewDecoder(r.Body).Decode(&req)

			require.Equal(t, 1, req.Current)
			require.Equal(t, -1, req.RowCount)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture(t, "unbound/searchHostOverride.json"))
		})

		got, err := client.ListHostOverrides(context.Background())
		require.NoError(t, err)

		want := []api.HostOverride{
			{
				ID:       "2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c",
				Hostname: "ha",
				Domain:   "home.yarotsky.me",
				Server:   "192.168.1.13",
			},
		}
		require.ElementsMatch(t, want, got)
	})
}

func TestCreateHostOverride(t *testing.T) {
	t.Run("creates a host override", func(t *testing.T) {
		client, teardown := setup(t)
		t.Cleanup(teardown)

		mux.HandleFunc("/api/unbound/settings/addHostOverride/", func(w http.ResponseWriter, r *http.Request) {
			var req api.HostOverrideRequest
			json.NewDecoder(r.Body).Decode(&req)

			require.Equal(t, "1", req.Host.Enabled)
			require.Equal(t, "ha", req.Host.Hostname)
			require.Equal(t, "home.yarotsky.me", req.Host.Domain)
			require.Equal(t, "A", req.Host.RR)
			require.Equal(t, "192.168.1.13", req.Host.Server)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture(t, "unbound/addHostOverride.json"))
		})

		rec, err := client.CreateHostOverride(context.Background(), api.HostOverride{
			Hostname: "ha",
			Domain:   "home.yarotsky.me",
			Server:   "192.168.1.13",
		})

		require.NoError(t, err)
		require.Equal(t, api.HostOverrideID("2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c"), rec.ID)
	})
}

func TestUpdateHostOverride(t *testing.T) {
	t.Run("updates a host override", func(t *testing.T) {
		client, teardown := setup(t)
		t.Cleanup(teardown)

		mux.HandleFunc("/api/unbound/settings/setHostOverride/59641e80-1f40-4d28-a7df-314c09c30800", func(w http.ResponseWriter, r *http.Request) {
			var req api.HostOverrideRequest
			json.NewDecoder(r.Body).Decode(&req)

			require.Equal(t, "1", req.Host.Enabled)
			require.Equal(t, "ha", req.Host.Hostname)
			require.Equal(t, "home.yarotsky.me", req.Host.Domain)
			require.Equal(t, "A", req.Host.RR)
			require.Equal(t, "192.168.1.13", req.Host.Server)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture(t, "unbound/setHostOverride.json"))
		})

		err := client.UpdateHostOverride(context.Background(), api.HostOverride{
			ID:       "59641e80-1f40-4d28-a7df-314c09c30800",
			Hostname: "ha",
			Domain:   "home.yarotsky.me",
			Server:   "192.168.1.13",
		})

		require.NoError(t, err)
	})
}

func TestDeleteHostOverride(t *testing.T) {
	t.Run("deletes a host override", func(t *testing.T) {
		client, teardown := setup(t)
		t.Cleanup(teardown)

		mux.HandleFunc("/api/unbound/settings/delHostOverride/2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c", func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			require.Equal(t, map[string]interface{}{}, req)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture(t, "unbound/delHostOverride.json"))
		})

		err := client.DeleteHostOverride(context.Background(), api.HostOverride{
			ID: "2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c",
		})

		require.NoError(t, err)
	})
}

func TestListHostAliases(t *testing.T) {
	t.Run("returns host aliases", func(t *testing.T) {
		client, teardown := setup(t)
		t.Cleanup(teardown)

		mux.HandleFunc("/api/unbound/settings/searchHostAlias/", func(w http.ResponseWriter, r *http.Request) {
			var req api.SearchHostAliasRequest
			json.NewDecoder(r.Body).Decode(&req)

			require.Equal(t, 1, req.Current)
			require.Equal(t, -1, req.RowCount)
			require.Equal(t, api.HostOverrideID("2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c"), req.HostID)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture(t, "unbound/searchHostAlias.json"))
		})

		got, err := client.ListHostAliases(context.Background(), api.HostOverrideID("2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c"))
		require.NoError(t, err)

		want := []api.HostAlias{
			{
				ID:       "18b07c57-fce4-43ad-8bd8-5fb0e8777800",
				Hostname: "test",
				Domain:   "home.yarotsky.me",
				Host:     "traefik.home.yarotsky.me",
				HostID:   api.HostOverrideID("2f0e73f7-fe3f-43fa-b8b0-fdf0ba48452c"),
			},
		}
		require.ElementsMatch(t, want, got)
	})
}

func TestCreateHostAlias(t *testing.T) {
	t.Run("creates a host alias", func(t *testing.T) {
		client, teardown := setup(t)
		t.Cleanup(teardown)

		mux.HandleFunc("/api/unbound/settings/addHostAlias/", func(w http.ResponseWriter, r *http.Request) {
			var req api.HostAliasRequest
			json.NewDecoder(r.Body).Decode(&req)

			require.Equal(t, "1", req.Alias.Enabled)
			require.Equal(t, "test2", req.Alias.Hostname)
			require.Equal(t, "home.yarotsky.me", req.Alias.Domain)
			require.Equal(t, api.HostOverrideID("a7a9f5ef-4ac1-4df4-bc8e-f122d02001ec"), req.Alias.HostID)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture(t, "unbound/addHostAlias.json"))
		})

		rec, err := client.CreateHostAlias(context.Background(), api.HostAlias{
			Hostname: "test2",
			Domain:   "home.yarotsky.me",
			HostID:   "a7a9f5ef-4ac1-4df4-bc8e-f122d02001ec",
		})

		require.NoError(t, err)
		require.Equal(t, api.HostAliasID("d7c20457-cad1-4ca2-afb4-7343354f0f1d"), rec.ID)
	})
}

func TestUpdateHostAlias(t *testing.T) {
	t.Run("updates a host alias", func(t *testing.T) {
		client, teardown := setup(t)
		t.Cleanup(teardown)

		mux.HandleFunc("/api/unbound/settings/setHostAlias/d7c20457-cad1-4ca2-afb4-7343354f0f1d", func(w http.ResponseWriter, r *http.Request) {
			var req api.HostAliasRequest
			json.NewDecoder(r.Body).Decode(&req)

			require.Equal(t, "1", req.Alias.Enabled)
			require.Equal(t, "test2", req.Alias.Hostname)
			require.Equal(t, "home.yarotsky.me", req.Alias.Domain)
			require.Equal(t, api.HostOverrideID("a7a9f5ef-4ac1-4df4-bc8e-f122d02001ec"), req.Alias.HostID)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture(t, "unbound/setHostAlias.json"))
		})

		err := client.UpdateHostAlias(context.Background(), api.HostAlias{
			ID:       "d7c20457-cad1-4ca2-afb4-7343354f0f1d",
			Hostname: "test2",
			Domain:   "home.yarotsky.me",
			HostID:   "a7a9f5ef-4ac1-4df4-bc8e-f122d02001ec",
		})

		require.NoError(t, err)
	})
}

func TestDeleteHostAlias(t *testing.T) {
	t.Run("deletes a host alias", func(t *testing.T) {
		client, teardown := setup(t)
		t.Cleanup(teardown)

		mux.HandleFunc("/api/unbound/settings/delHostAlias/d7c20457-cad1-4ca2-afb4-7343354f0f1d", func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			require.Equal(t, map[string]interface{}{}, req)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fixture(t, "unbound/delHostAlias.json"))
		})

		err := client.DeleteHostAlias(context.Background(), api.HostAlias{
			ID: "d7c20457-cad1-4ca2-afb4-7343354f0f1d",
		})

		require.NoError(t, err)
	})
}
