package repo

import (
	"fmt"
	"github.com/adrg/xdg"
	"log"
	"mvdan.cc/fdroidcl/adb"
	"os"
	"os/exec"
	"strings"
)

type GPlayCLIPackage struct {
	apk Apk
}

func (pkg GPlayCLIPackage) Apk() Apk {
	return pkg.apk
}

func (pkg GPlayCLIPackage) UpdateCache(device *adb.Device) (string, error) {
	apkDir, err := xdg.CacheFile("terraform-android/gplaycli")
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(apkDir, 0775)
	if err != nil {
		return "", err
	}

	apkPath := fmt.Sprintf("%s/%s.apk", apkDir, pkg)
	pkg.apk.Path = &apkPath

	cmd := exec.Command("python", "-m", "gplaycli")
	_, err = os.Stat(apkPath)
	if os.IsNotExist(err) {
		log.Println("Downloading", pkg)
		cmd.Args = append(cmd.Args, fmt.Sprint("--folder=", apkDir), fmt.Sprint("--download=", pkg))
	} else {
		log.Println("Updating cached packages")
		cmd.Args = append(cmd.Args, fmt.Sprint("--update=", apkDir), "--yes")
	}

	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if strings.Contains(string(stdouterr), "No module named gplaycli") {
		return "", fmt.Errorf("gplaycli is not installed (with this environment's `python`)")
	}
	if err != nil || strings.Contains(string(stdouterr), "[ERROR]") {
		return "", fmt.Errorf("Failed to download or update %s: %s", pkg, stdouterr)
	}
	log.Println(pkg, "cached")

	return *pkg.apk.Path, nil
}
