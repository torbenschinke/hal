package hue

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

//go:embed hue_ca.pem
var cacert []byte

func newClient(bridgeId string) *http.Client {
	// hue bridges currently do not support IP   but just
	// the CA name, which is not supported anymore, see also
	// https://golang.google.cn/doc/go1.15#commonname and
	// https://developers.meethue.com/develop/application-design-guidance/using-https/
	var caCertPool *x509.CertPool
	caCertPool = x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(cacert)

	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		ServerName: bridgeId,
		// CN was deprecated in 2000, and RFC 6125 made the fallback to it optional, rather a requirement.
		// Also, the root-ca is only valid until 2038, which is even from today only 16 years.
		// The according hardware and building will very likely exceed that lifetime.
		InsecureSkipVerify: true,
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Second * 15,
	}
}

func request[T any](bridge *Bridge, method string, path string, body any) (T, error) {
	var t T
	if len(bridge.Addresses) == 0 {
		return t, fmt.Errorf("no ip to connect")
	}

	client := bridge.client
	base := fmt.Sprintf("https://%s:%d", bridge.Addresses[0], bridge.Port)
	uri, err := url.JoinPath(base, path)
	if err != nil {
		return t, err
	}

	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return t, err
		}

		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, uri, reader)
	if err != nil {
		return t, err
	}

	if bridge.auth.username != "" {
		req.Header.Set("hue-application-key", bridge.auth.username)
	}

	resp, err := client.Do(req)
	if err != nil {
		return t, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return t, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// todo do we need the full buffering to snoop any 200 + error conditions?
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return t, fmt.Errorf("cannot read response body")
	}

	dec := json.NewDecoder(bytes.NewReader(buf))
	if err := dec.Decode(&t); err != nil {
		return t, fmt.Errorf("cannot decode json body: %w", err)
	}

	return t, nil
}

type clipv2[T any] struct {
	Errors ErrorList `json:"errors"`
	Data   []T       `json:"data"`
}
