package main

import (
	"context"
	"strings"

	"github.com/grandcat/zeroconf"
)

// DiscoveredDevice represents a device found via mDNS.
type DiscoveredDevice struct {
	Name string
	IP   string
	Port int
}

// StartDiscovery browses for Awair devices via mDNS and sends them on the
// returned channel. Browsing stops when the context is cancelled.
func StartDiscovery(ctx context.Context) <-chan DiscoveredDevice {
	ch := make(chan DiscoveredDevice)

	go func() {
		defer close(ch)

		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			return
		}

		entries := make(chan *zeroconf.ServiceEntry)

		go func() {
			for entry := range entries {
				name := strings.ToLower(entry.ServiceInstanceName())
				if !strings.HasPrefix(name, "awair") {
					continue
				}

				// Use the first IPv4 address
				if len(entry.AddrIPv4) == 0 {
					continue
				}
				ip := entry.AddrIPv4[0].String()

				select {
				case ch <- DiscoveredDevice{
					Name: entry.ServiceInstanceName(),
					IP:   ip,
					Port: entry.Port,
				}:
				case <-ctx.Done():
					return
				}
			}
		}()

		_ = resolver.Browse(ctx, "_http._tcp", "local.", entries)
	}()

	return ch
}
