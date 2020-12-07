package repo

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/adrg/xdg"
	"io"
	"io/ioutil"
	"log"
	"mvdan.cc/fdroidcl/adb"
	"mvdan.cc/fdroidcl/fdroid"
	"net/http"
	"os"
)

type FDroidPackage string

func (pkg FDroidPackage) Name() string {
	return string(pkg)
}

func (pkg FDroidPackage) Source() string {
	return "F-Droid"
}

func (pkg FDroidPackage) UpdateCache(device *adb.Device) (string, error) {
	jarpath, err := xdg.CacheFile("terraform-android/fdroid-index.jar")
	if err != nil {
		return "", err
	}

	log.Println("Downloading F-Droid index")
	if err = downloadEtag("https://f-droid.org/repo/index-v1.jar", jarpath, nil); err != nil && err != errNotModified {
		return "", err
	}

	jar, err := os.Open(jarpath)
	if err != nil {
		return "", err
	}

	stat, err := jar.Stat()
	if err != nil {
		return "", err
	}

	log.Println("Loading F-Droid index")
	index, err := fdroid.LoadIndexJar(jar, stat.Size(), nil)
	if err != nil {
		return "", err
	}

	var apk *fdroid.Apk
	for _, app := range index.Apps {
		log.Println("Found", app.PackageName)
		if app.PackageName == string(pkg) {
			if apk = app.SuggestedApk(device); apk == nil {
				return "", fmt.Errorf("No %s APK found for %s", pkg, device.Model)
			}
			break
		}
	}

	if apk == nil {
		return "", fmt.Errorf("No such %s app found", pkg)
	}

	if err := downloadEtag(apk.URL(), Path(pkg), apk.Hash); err != nil && err != errNotModified {
		return "", fmt.Errorf("Failed to download %s: %s", apk.ApkName, err)
	}

	return Path(pkg), nil
}

/* Borrowed from github.com/mvdan/fdroidcl/blob/4684bbe535147f80898e1e657bcd3cd253c11ec4/update.go
*   without modification under BSD-3 (unimportable since it's in `package main`).
 */
func respEtag(resp *http.Response) string {
	etags, e := resp.Header["Etag"]
	if !e || len(etags) == 0 {
		return ""
	}
	return etags[0]
}

var errNotModified = fmt.Errorf("not modified")
var httpClient = &http.Client{}

func downloadEtag(url, path string, sum []byte) error {
	fmt.Printf("Downloading %s... ", url)
	defer fmt.Println()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	etagPath := path + "-etag"
	if _, err := os.Stat(path); err == nil {
		etag, _ := ioutil.ReadFile(etagPath)
		req.Header.Add("If-None-Match", string(etag))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: %d %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	if resp.StatusCode == http.StatusNotModified {
		fmt.Printf("not modified")
		return errNotModified
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if sum == nil {
		_, err := io.Copy(f, resp.Body)
		if err != nil {
			return err
		}
	} else {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		got := sha256.Sum256(data)
		if !bytes.Equal(sum, got[:]) {
			return fmt.Errorf("sha256 mismatch")
		}
		if _, err := f.Write(data); err != nil {
			return err
		}
	}
	if err := ioutil.WriteFile(etagPath, []byte(respEtag(resp)), 0o644); err != nil {
		return err
	}
	fmt.Printf("done")
	return nil
}

/* END BORROW */
