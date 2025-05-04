package ctrl

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flytam/filenamify"
	"github.com/m1d1/go-fritz-backup/conf"
	"github.com/m1d1/go-fritz-backup/tr064"
)

type Controller struct {
	Settings       conf.Data
	Soap           *tr064.Soap
	FileNamePrefix string // Filename yyyy-mm-dd_FritzboxModel_Item.ext
}

func New() *Controller {
	return &Controller{}
}

func (c *Controller) ReadConfigFile() error {
	data, err := conf.New()
	if err == nil {
		c.Settings = data
		c.Soap = tr064.New(c.Settings.Device.URL, c.Settings.Device.Username, c.Settings.Device.Password)
	}
	return err
}

func (c *Controller) GetDeviceInfo() (string, error) {
	var msg string = ""
	res, err := c.Soap.Do("/upnp/control/deviceinfo", "urn:dslforum-org:service:DeviceInfo:1#GetInfo", "")
	if err != nil {
		return "", err
	}
	info := &tr064.XMLInfoResponse{}
	if err = xml.Unmarshal(res, &info); err != nil {
		return "", err
	}
	// fritzbox details
	msg = fmt.Sprintf("%s\nUp since: %s. (%s)\n", info.Body.GetInfoResponse.NewDescription,
		time.Now().Add(-(time.Duration(info.Body.GetInfoResponse.NewUpTime * int64(time.Second)))).Format("Monday 2006-01-02 15:04:05"), time.Duration(info.Body.GetInfoResponse.NewUpTime*int64(time.Second)))

	/////////////////
	// FILE PREFIX
	modelName := strings.ReplaceAll(info.Body.GetInfoResponse.NewModelName, "!", ".")
	modelName = strings.ReplaceAll(modelName, " ", "_")
	c.FileNamePrefix = fmt.Sprintf("%s_%s_%s", modelName, info.Body.GetInfoResponse.NewSoftwareVersion,
		time.Now().Format("02.01.06"))
	// time.Now().Format("2006-01-02"))

	return msg, nil
}

// downloadUrl, fileName, url, error
func (c *Controller) GetConfigFile() (string, string, *url.URL, error) {
	res, err := c.Soap.Do("/upnp/control/deviceconfig",
		"urn:dslforum-org:service:DeviceConfig:1#X_AVM-DE_GetConfigFile",
		fmt.Sprintf("<NewX_AVM-DE_Password>%s</NewX_AVM-DE_Password>", c.Settings.Export.Password))
	if err != nil {
		return "", "", nil, err
	}

	data := &tr064.XMLConfigFileResponse{}
	if err = xml.Unmarshal(res, &data); err != nil {
		return "", "", nil, err
	}
	// data.Body.XAVMDEGetConfigFileResponse.NewXAVMDEConfigFileUrl URL with new port not 49000
	u, err := url.Parse(data.Body.XAVMDEGetConfigFileResponse.NewXAVMDEConfigFileUrl)
	if err != nil {
		return "", "", nil, err
	}

	// Download .export file
	fname := fmt.Sprintf("%s-Config.export", c.FileNamePrefix)
	configDownloadURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	return configDownloadURL, fname, u, nil
}

func (c *Controller) BackupConfigFile(url *url.URL, configDownloadURL string, fname string) (string, error) {
	d := tr064.New(configDownloadURL, c.Soap.Username, c.Soap.Password)
	configBytes, err := d.Do(url.Path, "", "")
	if err != nil {
		return "", err
	}
	err = c.saveBytes(fname, configBytes)
	return fmt.Sprintf("Downloaded: %s", fname), err
}

func (c *Controller) BackupPhonebooks() (string, error) {
	var msg string = ""
	if !c.Settings.Export.Phonebooks {
		return "Backup phonebooks disabled", nil
	}
	// Get List of Phonebook Ids
	res, err := c.Soap.Do("/upnp/control/x_contact",
		"urn:dslforum-org:service:X_AVM-DE_OnTel:1#GetPhonebookList", "")
	if err != nil {
		return "", err
	}

	// Get coma separated list of phonebook IDs
	// <NewPhonebookList>0,1,2</NewPhonebookList>
	pbList := &tr064.XMLPhonebookListResponse{}
	if err = xml.Unmarshal(res, &pbList); err != nil {
		return "", err
	}

	pbListArr := strings.Split(pbList.Body.GetPhonebookListResponse.NewPhonebookList, ",")

	for _, pbID := range pbListArr {
		res, err = c.Soap.Do("/upnp/control/x_contact",
			"urn:dslforum-org:service:X_AVM-DE_OnTel:1#GetPhonebook", fmt.Sprintf("<NewPhonebookID>%s</NewPhonebookID>", pbID))
		if err != nil {
			return "", nil
		}
		pbInfo := &tr064.XMLPhonebookResponse{}
		if err = xml.Unmarshal(res, &pbInfo); err != nil {
			return "", err
		}
		// Convert a phonebook string to a valid safe filename
		fname := fmt.Sprintf("%s-Phonebook-%s", c.FileNamePrefix, pbInfo.Body.GetPhonebookResponse.NewPhonebookName)
		fname, err = filenamify.Filenamify(fname, filenamify.Options{
			Replacement: "_",
		})

		if err == nil {
			fname += ".xml" // add extension
			err = c.downloadFile(pbInfo.Body.GetPhonebookResponse.NewPhonebookURL, fname)
			if err != nil {
				msg += fmt.Sprintf("Failed to download: %s\n", pbInfo.Body.GetPhonebookResponse.NewPhonebookURL)
			} else {
				msg += fmt.Sprintf("Downloaded: %s\n", fname)
			}
		} else {
			msg += fmt.Sprintf("Failed to convert phonebook string to a safe filename: %s", err.Error())
		}
	}

	return msg, nil
}

func (c *Controller) DownloadCallBarringList() (string, error) {
	res, err := c.Soap.Do("/upnp/control/x_contact",
		"urn:dslforum-org:service:X_AVM-DE_OnTel:1#GetCallBarringList", "")
	if err != nil {
		return "", nil
	}

	blList := &tr064.XMLBarringListResponse{}
	if err = xml.Unmarshal(res, &blList); err != nil {
		return "", err
	}
	fname := fmt.Sprintf("%s-Barringlist.xml", c.FileNamePrefix)
	err = c.downloadFile(blList.Body.GetCallBarringListResponse.NewPhonebookURL, fname)

	return fmt.Sprintf("Downloaded: %s", fname), err
}

// PhoneAssets download not available via tr064 ?
// download it the old fashioned way - with a multipart form - port 80
func (c *Controller) GetAssetsFile() (string, error) {
	client := &http.Client{}

	// get a sid
	res, err := c.Soap.Do("/upnp/control/deviceconfig",
		"urn:dslforum-org:service:DeviceConfig:1#X_AVM-DE_CreateUrlSID", "")
	if err != nil {
		return "", err
	}
	SID := &tr064.XMLUrlSIDResponse{}
	if err = xml.Unmarshal(res, &SID); err != nil {
		return "", err
	}
	// extract id from string "sid=1234ab"
	sidID := strings.Split(SID.Body.XAVMDECreateUrlSIDResponse.NewXAVMDEUrlSID, "sid=")[1]
	ip, _, _ := net.SplitHostPort(c.Settings.Device.Host)
	requestUrl := fmt.Sprintf("http://%s/cgi-bin/firmwarecfg", ip)
	// The order of form-data key is mandatory: sid, AssetsImportExportPassword, AssetsExport
	var multipartMap []map[string]string // s[string]string
	multipartMap = append(multipartMap, map[string]string{"sid": sidID})
	multipartMap = append(multipartMap, map[string]string{"AssetsImportExportPassword": c.Settings.Export.Password})
	multipartMap = append(multipartMap, map[string]string{"AssetsExport": ""})

	var resp *http.Response
	multipartBuf := bytes.Buffer{}
	writer := multipart.NewWriter(&multipartBuf)

	// create multipart form-data
	for _, element := range multipartMap {
		for key, val := range element {
			if f, _ := writer.CreateFormField(key); err == nil {
				if _, err = f.Write([]byte(val)); err != nil {
					return "", err
				}
			} else {
				return "", err
			}
		}
	}
	err = writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, requestUrl, &multipartBuf)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Go-http-client/1.1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "*/*")

	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/zip;" {
		return "Download Phone Assets failed", nil
	}

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil {
		return "", err
	}

	err = c.saveBody(resp, params["filename"]) // #nosec G104 -- error (err) is checked by caller
	return fmt.Sprintf("Downloaded: %s", params["filename"]), err
}

func (c *Controller) downloadFile(url string, fname string) error {
	resp, err := http.Get(url) // #nosec G107 -- Url provided to HTTP request as taint input
	if err != nil {
		return err
	}
	return c.saveBody(resp, fname)
}

func (c *Controller) saveBody(resp *http.Response, fname string) error {
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	return c.saveBytes(fname, b)
}

func (c *Controller) saveBytes(fname string, b []byte) error {
	return os.WriteFile(filepath.Join(c.Settings.Backup.TargetPath, fname), b, 0o600)
}
