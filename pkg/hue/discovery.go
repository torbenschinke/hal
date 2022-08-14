package hue

import (
	"context"
	"fmt"
	"github.com/grandcat/zeroconf"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const dnsHueServiceName = "_hue._tcp"

type ClientKey struct {
	Username  string `json:"username"`
	ClientKey string `json:"clientkey"`
}

type Device struct {
	ID string `json:"id"`
}

type Bridge struct {
	ID        string   // bridgeid
	Model     string   // modelid
	Name      string   // service record instance
	Addresses []net.IP // ipv4 and ipv6
	Port      int
	client    *http.Client // keep instance for connection pooling
	auth      *struct {
		username string
	}
}

func (b Bridge) Config() (BridgeConfig, error) {
	return request[BridgeConfig](b, http.MethodGet, "/api/0/config", nil)
}

// GenerateClientKey requests a random key from the bridge.
// First, the user has to press the link button and
// then this call must be made to proof that the user is in control of the bridge.
// See also https://developers.meethue.com/develop/hue-api-v2/getting-started/.
func (b Bridge) GenerateClientKey(appname, instancename string) (ClientKey, error) {
	type rpcResponse struct {
		Success *ClientKey `json:"success"`
		Error   *Error     `json:"error"`
	}

	type generateClientKey struct {
		DeviceType        string `json:"devicetype"`
		GenerateClientKey bool   `json:"generateclientkey"`
	}

	res, err := request[[]rpcResponse](b, http.MethodPost, "/api", generateClientKey{
		DeviceType:        appname + "#" + instancename,
		GenerateClientKey: true,
	})

	if err != nil {
		return ClientKey{}, err
	}

	if len(res) == 0 {
		return ClientKey{}, fmt.Errorf("unexpected empty response")
	}

	if res[0].Error != nil {
		return ClientKey{}, res[0].Error
	}

	return *res[0].Success, err
}

// SetAuthentication requires the secret username (== token) of the one-time linkage with the bridge.
func (b Bridge) SetAuthentication(username string) {
	b.auth.username = username
}

func (b Bridge) Devices() ([]Device, error) {
	res, err := request[clipv2[Device]](b, http.MethodGet, "/clip/v2/resource/device", nil)
	if err != nil {
		return nil, err
	}

	if res.Errors != nil {
		return nil, res.Errors
	}

	return res.Data, nil
}

type BridgeConfig struct {
	Name             string `json:"name"`
	SoftwareVersion  string `json:"swversion"`
	ApiVersion       string `json:"apiversion"`
	Mac              string `json:"mac"`
	BridgeId         string `json:"bridgeid"`
	Factorynew       bool   `json:"factorynew"`
	ReplacesBridgeId any    `json:"replacesbridgeid"`
	ModelId          string `json:"modelid"`
}

func newDNSBridge(entry *zeroconf.ServiceEntry) Bridge {
	var bridge Bridge
	bridge.auth = &struct {
		username string
	}{}
	for _, text := range entry.Text {
		keyVal := strings.SplitN(text, "=", 2)
		if len(keyVal) != 2 {
			continue
		}

		switch keyVal[0] {
		case "bridgeid":
			bridge.ID = keyVal[1]
		case "modelid":
			bridge.Model = keyVal[1]
		}
	}

	bridge.client = newClient(bridge.ID)

	for _, ip := range entry.AddrIPv4 {
		bridge.Addresses = append(bridge.Addresses, ip)
	}

	for _, ip := range entry.AddrIPv6 {
		bridge.Addresses = append(bridge.Addresses, ip)
	}

	bridge.Port = entry.Port
	bridge.Name = entry.Instance

	return bridge
}

// Discover uses dns-sd to discover hue bridges.
// See also https://developers.meethue.com/develop/application-design-guidance/hue-bridge-discovery/.
func Discover(timeout time.Duration) ([]Bridge, error) {
	var bridges []Bridge
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			bridges = append(bridges, newDNSBridge(entry))
		}

		cancel()
	}(entries)

	err = resolver.Browse(ctx, dnsHueServiceName, "local.", entries)
	if err != nil {
		return bridges, fmt.Errorf("cannot browser hue bridges: %w", err)
	}

	<-ctx.Done()

	return bridges, nil
}
