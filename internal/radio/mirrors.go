package radio

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"
)

const MirrorDiscoveryHost = "all.api.radio-browser.info"

type mirrorResolver interface {
	LookupHost(context.Context, string) ([]string, error)
	LookupAddr(context.Context, string) ([]string, error)
}

// DiscoverMirrors follows Radio Browser's documented DNS discovery. A hostname
// announced by DNS is not a guarantee that its HTTP service is available.
func DiscoverMirrors(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	return discoverMirrors(ctx, net.DefaultResolver)
}

func discoverMirrors(ctx context.Context, resolver mirrorResolver) ([]string, error) {
	addresses, err := resolver.LookupHost(ctx, MirrorDiscoveryHost)
	if err != nil {
		return nil, fmt.Errorf("descubrir espejos: %w", err)
	}
	mirrors := []string{}
	var lookupErr error
	for _, address := range addresses {
		names, err := resolver.LookupAddr(ctx, address)
		if err != nil {
			lookupErr = err
			continue
		}
		for _, name := range names {
			name = strings.TrimSuffix(strings.ToLower(name), ".")
			if !strings.HasSuffix(name, ".api.radio-browser.info") || strings.ContainsAny(name, "/:@?# ") {
				continue
			}
			mirror := "https://" + name
			if !contains(mirrors, mirror) {
				mirrors = append(mirrors, mirror)
			}
		}
	}
	if len(mirrors) == 0 {
		if lookupErr != nil {
			return nil, fmt.Errorf("resolver nombres de espejos: %w", lookupErr)
		}
		return nil, fmt.Errorf("DNS no anunció espejos de Radio Browser")
	}
	sort.Strings(mirrors)
	return mirrors, nil
}
