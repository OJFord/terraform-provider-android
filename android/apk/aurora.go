package repo

import (
	"fmt"
	"log"
	"os"
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

	log.Printf("[DEBUG] AuroraStore to download %s", pkg.apk.Name)
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
		return "", fmt.Errorf("Failed to trigger download for %s: is `com.aurora.store.debug` installed?", pkg.apk.Name)
	}
	if err != nil {
		return "", fmt.Errorf("Failed to trigger download for %s: %s", pkg.apk.Name, stdouterr)
	}

	auroraPkgDir := fmt.Sprintf("sdcard/Aurora/Store/Downloads/%s", pkg.apk.Name)
	// We need to delete first in order to identify download has happened
	// TODO: Think of a better way.. maybe aurora `touch`es latest version even if it doesn't need to download?
	cmd = device.AdbCmd("shell", "rm", "-rf", auroraPkgDir)
	stdouterr, err = cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return "", fmt.Errorf("Failed to remove old versions of %s: %s", pkg.apk.Name, err)
	}

	var stdout []byte
	for !strings.Contains(string(stdout), pkg.apk.Name) {
		time.Sleep(3 * time.Second)
		cmd = device.AdbCmd("shell", "ls", "sdcard/Aurora/Store/Downloads/")
		stdout, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}

	cmd = device.AdbCmd("shell", "ls", auroraPkgDir)
	stdout, err = cmd.Output()
	if err != nil {
		return "", err
	}
	versionDownloaded := strings.TrimSpace(string(stdout))

	log.Printf("[INFO] Downloading %s @ %s", pkg.apk.Name, string(versionDownloaded))
	// Agh, it also creates the directories & APK immediately, while still downloading...
	// TODO: work out something better than just waiting 'should be long enough'
	time.Sleep(60 * time.Second)

	cmd = device.AdbCmd("pull", auroraPkgDir, fmt.Sprintf("%s/", apkDir))
	stdouterr, err = cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve %s: %s", pkg.apk.Name, stdouterr)
	}

	apkPath := fmt.Sprintf("%s/%s/%s/%s.apk", apkDir, pkg.apk.Name, string(versionDownloaded), pkg.apk.Name)
	pkg.apk.Path = &apkPath
	return *pkg.apk.Path, nil
}
