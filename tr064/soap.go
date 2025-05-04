package tr064

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/icholy/digest"
)

type Soap struct {
	BaseURL  string
	Username string
	Password string
	Client   *http.Client
	Request  *http.Request
	Response *http.Response
}

// Fritz DeviceInfo
type XMLInfoResponse struct {
	XMLName       xml.Name `xml:"Envelope"`
	Text          string   `xml:",chardata"`
	S             string   `xml:"s,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          struct {
		Text            string `xml:",chardata"`
		GetInfoResponse struct {
			Text string `xml:",chardata"`
			U    string `xml:"u,attr"`
			// NewManufacturerName string `xml:"NewManufacturerName"`
			// NewManufacturerOUI  string `xml:"NewManufacturerOUI"`
			NewModelName   string `xml:"NewModelName"`
			NewDescription string `xml:"NewDescription"`
			// NewProductClass     string `xml:"NewProductClass"`
			// NewSerialNumber     string `xml:"NewSerialNumber"`
			NewSoftwareVersion string `xml:"NewSoftwareVersion"`
			// NewHardwareVersion  string `xml:"NewHardwareVersion"`
			// NewSpecVersion      string `xml:"NewSpecVersion"`
			// NewProvisioningCode string `xml:"NewProvisioningCode"`
			NewUpTime int64 `xml:"NewUpTime"`
			// NewDeviceLog        string `xml:"NewDeviceLog"`
		} `xml:"GetInfoResponse"`
	} `xml:"Body"`
}

// Configuration Export
type XMLConfigFileResponse struct {
	XMLName       xml.Name `xml:"Envelope"`
	Text          string   `xml:",chardata"`
	S             string   `xml:"s,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          struct {
		Text                        string `xml:",chardata"`
		XAVMDEGetConfigFileResponse struct {
			Text                   string `xml:",chardata"`
			U                      string `xml:"u,attr"`
			NewXAVMDEConfigFileUrl string `xml:"NewX_AVM-DE_ConfigFileUrl"`
		} `xml:"X_AVM-DE_GetConfigFileResponse"`
	} `xml:"Body"`
}

type XMLPhonebookListResponse struct {
	XMLName       xml.Name `xml:"Envelope"`
	Text          string   `xml:",chardata"`
	S             string   `xml:"s,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          struct {
		Text                     string `xml:",chardata"`
		GetPhonebookListResponse struct {
			Text             string `xml:",chardata"`
			U                string `xml:"u,attr"`
			NewPhonebookList string `xml:"NewPhonebookList"`
		} `xml:"GetPhonebookListResponse"`
	} `xml:"Body"`
}

type XMLPhonebookResponse struct {
	XMLName       xml.Name `xml:"Envelope"`
	Text          string   `xml:",chardata"`
	S             string   `xml:"s,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          struct {
		Text                 string `xml:",chardata"`
		GetPhonebookResponse struct {
			Text                string `xml:",chardata"`
			U                   string `xml:"u,attr"`
			NewPhonebookName    string `xml:"NewPhonebookName"`
			NewPhonebookExtraID string `xml:"NewPhonebookExtraID"`
			NewPhonebookURL     string `xml:"NewPhonebookURL"`
		} `xml:"GetPhonebookResponse"`
	} `xml:"Body"`
}

type XMLBarringListResponse struct {
	XMLName       xml.Name `xml:"Envelope"`
	Text          string   `xml:",chardata"`
	S             string   `xml:"s,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          struct {
		Text                       string `xml:",chardata"`
		GetCallBarringListResponse struct {
			Text            string `xml:",chardata"`
			U               string `xml:"u,attr"`
			NewPhonebookURL string `xml:"NewPhonebookURL"`
		} `xml:"GetCallBarringListResponse"`
	} `xml:"Body"`
}

type XMLUrlSIDResponse struct {
	XMLName       xml.Name `xml:"Envelope"`
	Text          string   `xml:",chardata"`
	S             string   `xml:"s,attr"`
	EncodingStyle string   `xml:"encodingStyle,attr"`
	Body          struct {
		Text                       string `xml:",chardata"`
		XAVMDECreateUrlSIDResponse struct {
			Text            string `xml:",chardata"`
			U               string `xml:"u,attr"`
			NewXAVMDEUrlSID string `xml:"NewX_AVM-DE_UrlSID"`
		} `xml:"X_AVM-DE_CreateUrlSIDResponse"`
	} `xml:"Body"`
}

// user: foo,
// pass: bar
// baseURL: http://192.168.0.1
func New(baseURL, username, password string) *Soap {
	return &Soap{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // #nosec G402  --  false positive, x509: cannot validate certificate for 192.168.0.1 because it doesn't contain any IP SANs
				},
			},
		},
		Request:  nil,
		Response: nil,
	}
}

func (s *Soap) Do(urlPath string, soapAction string, extraPayload string) ([]byte, error) {
	payloadXML, err := s.newPayload(soapAction, extraPayload)
	if err != nil {
		return nil, err
	}

	if err := s.newSoapRequest(urlPath, payloadXML, soapAction); err != nil {
		return nil, err
	}

	if s.Response, err = s.Client.Do(s.Request); err != nil {
		return nil, err
	}

	// Expect Unauthorized
	if s.Response.StatusCode == 200 {
		slog.Info("No authentication needed?")
		return s.ReadResponseBody()

	} else if s.Response.StatusCode != 401 {
		return nil, fmt.Errorf("http error: %s", s.Response.Status)
	}

	chal, _ := digest.ParseChallenge(s.Response.Header.Get("WWW-Authenticate"))

	// use it to create credentials for the next request
	cred, err := digest.Digest(chal, digest.Options{
		Username: s.Username,
		Password: s.Password,
		Method:   s.Request.Method,
		URI:      s.Request.URL.RequestURI(),
		GetBody:  s.Request.GetBody,
		Count:    1,
	})
	if err != nil {
		return nil, err
	}
	if err := s.newSoapRequest(urlPath, payloadXML, soapAction); err != nil {
		return nil, err
	}

	s.Request.Header.Set("Authorization", cred.String())
	if s.Response, err = s.Client.Do(s.Request); err != nil {
		return nil, err
	}
	return s.ReadResponseBody()
}

func (s *Soap) newSoapRequest(urlPath string, payload []byte, soapAction string) error {
	url, err := url.Parse(s.BaseURL)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, urlPath)

	s.Request, err = http.NewRequest(http.MethodPost, url.String(), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	s.Request.Header.Add("Content-Type", "text/xml; charset=utf-8")
	// add non-empty soapAction header
	if soapAction != "" {
		s.Request.Header.Add("SoapAction", soapAction)
	}

	return nil
}

// add soapAction header if defined
func (s *Soap) ReadResponseBody() ([]byte, error) {
	bodyBytes, err := io.ReadAll(s.Response.Body)
	if err != nil {
		return nil, err
	}
	err = s.Response.Body.Close()
	return bodyBytes, err
}

// Extract SoapAction parts and paste it in the payload
// Split by #. Expected two parts
// ie. urn:dslforum-org:service:X_AVM-DE_OnTel:1#GetCallList
func (s *Soap) newPayload(soapAction string, optionalXMLPayload string) ([]byte, error) {
	// no soapAction = no payload. not an error
	if soapAction == "" {
		return []byte{}, nil
	}
	parts := strings.Split(soapAction, "#")
	if len(parts) != 2 {
		return []byte(""), fmt.Errorf("cannot generated XML payload from SoapAction")
	}
	return []byte(fmt.Sprintf(
		`<?xml version="1.0" encoding="utf-8" ?>
	<s:Envelope s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
	    <s:Body>
	        <u:%s xmlns:u="%s" >
			%s
			</u:%s>			
	    </s:Body>
	</s:Envelope>`, parts[1], parts[0], optionalXMLPayload, parts[1])), nil
}
