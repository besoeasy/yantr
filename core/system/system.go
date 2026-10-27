// Package system provides system-level utilities: IP identity, architecture detection.
package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"core/shared"
)

// IPIdentity holds information about the public IP address. Only the fields
// MachineIdentityCard.vue renders are kept: ip, city, region, country, isp/org
// — plus Source and FetchedAt, which say which provider answered and when, and
// CacheTTLMs, which states the server-side cache contract. Provider extras the
// card never showed (lat/lng, ASN, timezone, country code) were dropped with
// the presence feed.
type IPIdentity struct {
	IP         string `json:"ip"`
	City       string `json:"city,omitempty"`
	Region     string `json:"region,omitempty"`
	Country    string `json:"country,omitempty"`
	ISP        string `json:"isp,omitempty"`
	Org        string `json:"org,omitempty"`
	Source     string `json:"source"`
	FetchedAt  string `json:"fetchedAt"`
	CacheTTLMs int64  `json:"cacheTtlMs"`
}

const ipCacheTTL = 5 * time.Minute

var (
	ipMu       sync.Mutex
	ipCache    *IPIdentity
	ipCacheExp time.Time
)

// GetPublicIPIdentityCached returns the public IP identity, cached for 5 minutes.
func GetPublicIPIdentityCached(forceRefresh bool) (*IPIdentity, error) {
	ipMu.Lock()
	defer ipMu.Unlock()

	if !forceRefresh && ipCache != nil && time.Now().Before(ipCacheExp) {
		return ipCache, nil
	}

	identity, err := fetchPublicIP()
	if err != nil {
		return nil, err
	}
	ipCache = identity
	ipCacheExp = time.Now().Add(ipCacheTTL)
	return identity, nil
}

type provider struct {
	name string
	url  string
}

var providers = []provider{
	{"ipwho.is", "https://ipwho.is/"},
	{"ipapi.co", "https://ipapi.co/json/"},
	{"ipinfo.io", "https://ipinfo.io/json"},
	{"ifconfig.co", "https://ifconfig.co/json"},
}

func fetchPublicIP() (*IPIdentity, error) {
	client := &http.Client{Timeout: 6 * time.Second}
	fetchedAt := time.Now().UTC().Format(time.RFC3339)

	var errs []string
	for _, p := range providers {
		identity, err := fetchFromProvider(client, p, fetchedAt)
		if err != nil {
			errs = append(errs, p.name+": "+err.Error())
			continue
		}
		if identity != nil && identity.IP != "" {
			return identity, nil
		}
		errs = append(errs, p.name+": invalid response")
	}
	return nil, fmt.Errorf("failed to resolve public IP (%s)", strings.Join(errs, "; "))
}

func fetchFromProvider(client *http.Client, p provider, fetchedAt string) (*IPIdentity, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", p.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "yantr")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	return normalizeIPResponse(p.name, raw, fetchedAt), nil
}

func normalizeIPResponse(source string, raw map[string]interface{}, fetchedAt string) *IPIdentity {
	getString := func(key string) string {
		if v, ok := raw[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	id := &IPIdentity{
		Source:     source,
		FetchedAt:  fetchedAt,
		CacheTTLMs: int64(ipCacheTTL / time.Millisecond),
	}

	switch source {
	case "ipwho.is":
		if success, _ := raw["success"].(bool); !success {
			return nil
		}
		id.IP = getString("ip")
		id.City = getString("city")
		id.Region = getString("region")
		id.Country = getString("country")
		if conn, ok := raw["connection"].(map[string]interface{}); ok {
			if isp, ok := conn["isp"].(string); ok {
				id.ISP = isp
			}
			if org, ok := conn["org"].(string); ok {
				id.Org = org
			}
		}

	case "ipapi.co":
		if errMsg, _ := raw["error"].(bool); errMsg {
			return nil
		}
		id.IP = getString("ip")
		id.City = getString("city")
		id.Region = getString("region")
		id.Country = getString("country_name")
		id.ISP = getString("org")
		id.Org = getString("org")

	case "ipinfo.io":
		id.IP = getString("ip")
		id.City = getString("city")
		id.Region = getString("region")
		id.Country = getString("country")
		id.ISP = getString("org")
		id.Org = getString("org")

	case "ifconfig.co":
		id.IP = getString("ip")
		id.City = getString("city")
		id.Region = shared.Coalesce(getString("region_name"), getString("region"))
		id.Country = getString("country")
		id.ISP = shared.Coalesce(getString("asn_org"), getString("organization"))
		id.Org = shared.Coalesce(getString("asn_org"), getString("organization"))
	}

	return id
}
