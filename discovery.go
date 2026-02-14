package main

import (
	"context"
	"io"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

// DiscoveredDevice represents a device found via mDNS.
type DiscoveredDevice struct {
	Name string
	IP   string
	Port int
}

// StartDiscovery queries for Awair devices via mDNS and sends them on the
// returned channel. It re-queries every 30 seconds until the context is
// cancelled, to catch devices that come online later.
func StartDiscovery(ctx context.Context) <-chan DiscoveredDevice {
	ch := make(chan DiscoveredDevice)

	go func() {
		defer close(ch)

		// Suppress hashicorp/mdns log noise (IPv6 bind errors, client close info)
		log.SetOutput(io.Discard)

		for {
			entries := make(chan *mdns.ServiceEntry, 16)

			go func() {
				for entry := range entries {
					name := strings.ToLower(entry.Name)
					if !strings.Contains(name, "awair") {
						continue
					}

					if entry.AddrV4 == nil {
						continue
					}

					// entry.Name is "AWAIR-ELEM-XXXXX._http._tcp.local."
					// Extract the instance name (before the service type)
					instanceName := entry.Name
					if idx := strings.Index(instanceName, "._http._tcp"); idx > 0 {
						instanceName = instanceName[:idx]
					}

					select {
					case ch <- DiscoveredDevice{
						Name: instanceName,
						IP:   entry.AddrV4.String(),
						Port: entry.Port,
					}:
					case <-ctx.Done():
						return
					}
				}
			}()

			params := mdns.DefaultParams("_http._tcp")
			params.Entries = entries
			params.Timeout = 5 * time.Second
			_ = mdns.Query(params)
			close(entries)

			// Wait before re-querying, or exit if context is done
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
			}
		}
	}()

	return ch
}
