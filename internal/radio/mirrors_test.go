package radio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
)

type fakeMirrorResolver struct {
	addresses []string
	names     map[string][]string
	err       error
}

func (r fakeMirrorResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	if host != MirrorDiscoveryHost {
		return nil, fmt.Errorf("unexpected discovery host: %s", host)
	}
	return r.addresses, r.err
}
func (r fakeMirrorResolver) LookupAddr(ctx context.Context, address string) ([]string, error) {
	return r.names[address], nil
}

func TestDiscoverListsAllMirrorsAndDeduplicatesAddresses(t *testing.T) {
	r := fakeMirrorResolver{names: map[string][]string{}}
	for i := 0; i < 12; i++ {
		address := fmt.Sprintf("192.0.2.%d", i+1)
		r.addresses = append(r.addresses, address)
		r.names[address] = []string{fmt.Sprintf("mirror%02d.api.radio-browser.info.", i), "unrelated.example."}
	}
	r.addresses = append(r.addresses, "2001:db8::1")
	r.names["2001:db8::1"] = r.names["192.0.2.1"]
	mirrors, err := discoverMirrors(context.Background(), r)
	if err != nil || len(mirrors) != 12 || mirrors[0] != "https://mirror00.api.radio-browser.info" {
		t.Fatalf("mirrors=%v err=%v", mirrors, err)
	}
}

func TestDiscoverDoesNotInventMirrors(t *testing.T) {
	r := fakeMirrorResolver{addresses: []string{"192.0.2.1"}, names: map[string][]string{"192.0.2.1": {"de1.api.radio-browser.info."}}}
	mirrors, err := discoverMirrors(context.Background(), r)
	if err != nil || len(mirrors) != 1 {
		t.Fatalf("mirrors=%v err=%v", mirrors, err)
	}
	_, err = discoverMirrors(context.Background(), fakeMirrorResolver{err: context.Canceled})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestDiscoverMirrorsLive(t *testing.T) {
	if os.Getenv("AURA_RADIO_LIVE_TEST") != "1" {
		t.Skip("set AURA_RADIO_LIVE_TEST=1")
	}
	mirrors, err := DiscoverMirrors(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("DNS announced %d mirror(s): %v; HTTP availability not checked", len(mirrors), mirrors)
}
