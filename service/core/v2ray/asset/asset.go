package asset

import (
	"fmt"
	"github.com/muhammadmuzzammil1998/jsonc"
	"github.com/xbclub/xraya/common/files"
	"github.com/xbclub/xraya/conf"
	"github.com/xbclub/xraya/pkg/util/log"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

func GetV2rayLocationAssetOverride() string {
	return conf.GetEnvironmentConfig().Config;
}

func GetV2rayLocationAsset(filename string) (string, error) {
	return filepath.Join(conf.GetEnvironmentConfig().Config, filename), nil
}

func DoesV2rayAssetExist(filename string) bool {
	fullpath, err := GetV2rayLocationAsset(filename)
	if err != nil {
		return false
	}
	_, err = os.Stat(fullpath)
	if err != nil {
		return false
	}
	return true
}

func GetGFWListModTime() (time.Time, error) {
	fullpath, err := GetV2rayLocationAsset("LoyalsoldierSite.dat")
	if err != nil {
		return time.Now(), err
	}
	return files.GetFileModTime(fullpath)
}
func IsCustomExists() bool {
	return DoesV2rayAssetExist("custom.dat")
}

func GetConfigBytes() (b []byte, err error) {
	b, err = os.ReadFile(GetV2rayConfigPath())
	if err != nil {
		log.Warn("failed to get config: %v", err)
		return
	}
	b = jsonc.ToJSON(b)
	return
}

func GetV2rayConfigPath() (p string) {
	return path.Join(conf.GetEnvironmentConfig().Config, "config.json")
}

func GetV2rayConfigDirPath() (p string) {
	return conf.GetEnvironmentConfig().V2rayConfigDirectory
}

func Download(url string, to string) (err error) {
	log.Info("Downloading %v to %v", url, to)
	c := http.Client{Timeout: 90 * time.Second}
	resp, err := c.Get(url)
	if err != nil || resp.StatusCode != 200 {
		if err == nil {
			defer resp.Body.Close()
			err = fmt.Errorf("code: %v %v", resp.StatusCode, resp.Status)
		}
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(to, b, 0644)
}
