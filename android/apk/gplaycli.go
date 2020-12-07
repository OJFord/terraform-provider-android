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

type GPlayCLIPackage string

func (pkg GPlayCLIPackage) Name() string {
	return string(pkg)
}

func (pkg GPlayCLIPackage) Source() string {
	return "Google Play Store via gplaycli"
}

func (pkg GPlayCLIPackage) UpdateCache(device *adb.Device) (string, error) {
	apk_dir, err := xdg.CacheFile("terraform-android/")
	if err != nil {
		return "", err
	}

	cmd := exec.Command("python", "-m", "gplaycli")
	if device.Device != "" {
		cmd.Args = append(cmd.Args, fmt.Sprint("--device-codename=", device.Device))
	}

	_, err = os.Stat(fmt.Sprint(apk_dir, "/", pkg, ".apk"))
	if os.IsNotExist(err) {
		log.Println("Downloading", pkg)
		cmd.Args = append(cmd.Args, fmt.Sprint("--folder=", apk_dir), fmt.Sprint("--download=", pkg))
	} else {
		log.Println("Updating cached packages")
		cmd.Args = append(cmd.Args, fmt.Sprint("--update=", apk_dir), "--yes")
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

	return fmt.Sprint(apk_dir, "/", pkg, ".apk"), nil
}
