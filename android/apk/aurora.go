package repo

import (
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"mvdan.cc/fdroidcl/adb"

	_ "embed"
)

//go:embed AuroraStore/app/build/outputs/apk/debug/app-debug.apk
var comAuroraStoreApk []byte

type AuroraPackage struct {
	apk *Apk
}

func (pkg AuroraPackage) Apk() *Apk {
	return pkg.apk
}

func (pkg AuroraPackage) triggerDownload(device *adb.Device) error {
	log.Printf("[DEBUG] Requested AuroraStore to download %s", pkg.apk.Name)
	cmd := device.AdbCmd(
		"shell",
		"am",
		"start",
		"-n", "com.aurora.store.debug/com.aurora.store.view.ui.details.AppDetailsActivity",
		"-d", fmt.Sprintf("market://?id=%s\\&download", pkg.apk.Name),
	)

	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if strings.Contains(string(stdouterr), "Activity class {com.aurora.store.debug/com.aurora.store.view.ui.details.AppDetailsActivity} does not exist") {
		return fmt.Errorf("Failed to trigger download for %s: is `com.aurora.store.debug` installed?", pkg.apk.Name)
	}
	if err != nil {
		return fmt.Errorf("Failed to trigger download for %s: %s", pkg.apk.Name, stdouterr)
	}

	return nil
}

func (pkg AuroraPackage) UpdateCache(device *adb.Device) (string, error) {
	apkDir, err := xdg.CacheFile("terraform-android/aurora")
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(apkDir, 0775)
	if err != nil {
		return "", err
	}

	if pkg.apk.Name == "com.aurora.store.debug" {
		log.Println("[DEBUG] Bootstrapping AuroraStore")
		apkPath := fmt.Sprintf("%s/%s.apk", apkDir, pkg.apk.Name)
		pkg.apk.Path = &apkPath
		if err = os.WriteFile(apkPath, comAuroraStoreApk, 0666); err != nil {
			log.Println("[ERROR] Failed to bootstrap AuroraStore")
			return "", err
		}
		return *pkg.apk.Path, nil
	}

	err = pkg.triggerDownload(device)
	if err != nil {
		return "", err
	}

	auroraPkgDir := fmt.Sprintf("sdcard/Aurora/Store/Downloads/%s", pkg.apk.Name)
	downloadMarkers := fmt.Sprintf("%s/.*.download-*", auroraPkgDir)

	var stdout []byte
	for i := 2; strings.Contains(string(stdout), "download-in-progress") || !strings.Contains(string(stdout), "download-complete"); i++ {
		if i >= 6 {
			i = 0
			err = pkg.triggerDownload(device)
			if err != nil {
				return "", err
			}
		}
		time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)

		// || true to handle dir not existing, or no download markers existing yet
		cmd := device.AdbCmd("shell", "ls", "-A1t", downloadMarkers, "||", "true")
		stdout, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}

	re := regexp.MustCompile(`.(?P<version>[0-9]+).download-complete`)
	matches := re.FindSubmatch(stdout)
	index := re.SubexpIndex("version")
	versionDownloaded := string(matches[index])
	log.Printf("[INFO] Downloaded %s @ %s", pkg.apk.Name, versionDownloaded)

	cmd := device.AdbCmd("pull", auroraPkgDir, fmt.Sprintf("%s/", apkDir))
	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve %s: %s", pkg.apk.Name, stdouterr)
	}

	apkPath := fmt.Sprintf("%s/%s/%s/%s.apk", apkDir, pkg.apk.Name, versionDownloaded, pkg.apk.Name)
	pkg.apk.Path = &apkPath
	return *pkg.apk.Path, nil
}
