package fritzbox

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"golang.org/x/text/encoding/unicode"
	"io"
	"net/http"
	"net/url"
	"time"
)

type WLANDevice struct {
	UID  string `json:"uid"`  // not a UUID, e.g. landevice3812
	Name string `json:"name"` // e.g. iPhonevonTorben
	Type string `json:"type"` // e.g. passive or active
}

// loginResponse looks like
type loginResponse struct {
	SID       string `xml:"SID"`
	Challenge string `xml:"Challenge"`
	BlockTime string `xml:"BlockTime"`
}

// Service provides access to the fritzbox using the unofficial REST api. See also
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID.pdf
type Service struct {
	Host   string
	Client *http.Client
	SID    string
}

func NewService() *Service {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // there seems to be no spec to validate or pin the fb root certificate
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * 15,
	}

	return &Service{Host: "fritz.box", Client: client}
}

func (s *Service) SessionID(username, password string) (string, error) {
	res, err := s.Client.Get(fmt.Sprintf("https://%s/login_sid.lua", s.Host))
	if err != nil {
		return "", fmt.Errorf("cannot grab initial sid: %w", err)
	}

	var loginResp loginResponse
	if err := parseXML(res.Body, &loginResp); err != nil {
		return "", err
	}

	if loginResp.SID != "0000000000000000" {
		return loginResp.SID, nil
	}

	response := calcResponse(loginResp.Challenge, password)
	res, err = s.Client.Get(fmt.Sprintf("https://%s/login_sid.lua?username=%s&response=%s", s.Host, username, response))
	if err != nil {
		return "", fmt.Errorf("cannot finish challenge: %w", err)
	}

	loginResp = loginResponse{}
	if err := parseXML(res.Body, &loginResp); err != nil {
		return "", err
	}

	if loginResp.SID == "0000000000000000" {
		return "", fmt.Errorf("invalid credentials, blocked %s", loginResp.BlockTime)
	}

	return loginResp.SID, nil
}

func (s *Service) WLANDevices() ([]WLANDevice, error) {
	values := url.Values{}
	values.Add("sid", s.SID)
	//values.Add("xhrId", "wlanDevices")
	values.Add("page", "wSet")
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s/data.lua", s.Host), bytes.NewReader([]byte(values.Encode())))
	if err != nil {
		panic(fmt.Errorf("illegal state: %w", err))
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot get data: %w", err)
	}

	defer resp.Body.Close()

	type body struct {
		Data struct {
			WlanSettings struct {
				KnownWlanDevices []WLANDevice `json:"knownWlanDevices"`
			} `json:"wlanSettings"`
		} `json:"data"`
	}
	var res body
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&res); err != nil {
		return nil, fmt.Errorf("cannot decode response: %w", err)
	}

	return res.Data.WlanSettings.KnownWlanDevices, nil
}

func calcResponse(challenge, password string) string {
	return challenge + "-" + md5Hash(challenge+"-"+password)
}

func md5Hash(text string) string {
	enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	buf, err := enc.NewEncoder().Bytes([]byte(text))
	if err != nil {
		panic(fmt.Errorf("cannot fail: %w", err))
	}
	hsum := md5.Sum(buf)
	return hex.EncodeToString(hsum[:])
}

func parseXML(r io.ReadCloser, i interface{}) error {
	defer r.Close()

	dec := xml.NewDecoder(r)
	if err := dec.Decode(i); err != nil {
		if err != nil {
			return fmt.Errorf("cannot read xml response: %w", err)
		}
	}

	return nil
}
