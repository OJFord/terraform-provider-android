package android

import (
	"fmt"
	"github.com/adrg/xdg"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func resourceAndroidApk() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an Android APK resource. This can be used to create, read, update, and delete installed APKs (apps) on your Android device.",
		Create:      resourceAndroidApkCreate,
		Read:        resourceAndroidApkRead,
		Update:      resourceAndroidApkUpdate,
		Delete:      resourceAndroidApkDelete,

		Schema: map[string]*schema.Schema{
			"adb_serial": {
				Description: "Serial number (`get-serialno`) of the device - e.g. <IP>:<PORT> for TCP/IP devices",
				Required:    true,
				Type:        schema.TypeString,
			},
			"device_codename": {
				Default:     "",
				Description: "Device codename to present to Play Store (may affect app availability, or more?)",
				Optional:    true,
				Type:        schema.TypeString,
			},
			"name": {
				Description: "Qualified name of the package to install, e.g. `com.google.zxing.client.android`",
				ForceNew:    true,
				Required:    true,
				Type:        schema.TypeString,
			},
			"version": {
				Description: "Monotonically increasing `versionCode` of the package, safe for comparison",
				Computed:    true,
				Type:        schema.TypeInt,
			},
			"version_name": {
				Description: "Human-friendly `versionName`, defined by the package author and not guaranteed to increment",
				Computed:    true,
				Type:        schema.TypeString,
			},
		},

		CustomizeDiff: customiseDiff,
	}
}

func customiseDiff(diff *schema.ResourceDiff, _ interface{}) error {
	pkg := diff.Get("name").(string)
	device_codename := diff.Get("device_codename").(string)

	v, err := getLatestVersion(pkg, device_codename)
	if err != nil {
		return err
	}

	err = diff.SetNew("version", v)
	if err != nil {
		return err
	}

	vold, vnew := diff.GetChange("version")
	if vold.(int) > vnew.(int) {
		diff.ForceNew("version")
	}

	vn, err := getLatestVersionName(pkg, device_codename)
	if err != nil {
		return err
	}

	err = diff.SetNew("version_name", vn)
	if err != nil {
		return err
	}

	return nil
}

func updateCachedApk(pkg string, device_codename string) (string, error) {
	apk_dir, err := xdg.CacheFile("terraform-android/")
	if err != nil {
		return "", err
	}

	cmd := exec.Command("python", "-m", "gplaycli")
	if device_codename != "" {
		cmd.Args = append(cmd.Args, fmt.Sprint("--device-codename=", device_codename))
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
	if err != nil {
		return "", err
	}
	if strings.Contains(string(stdouterr), "[ERROR]") {
		return "", fmt.Errorf("Failed to download or update %s", pkg)
	}
	log.Println(pkg, "cached")

	return fmt.Sprint(apk_dir, "/", pkg, ".apk"), nil
}

func getLatestVersion(pkg string, device_codename string) (int, error) {
	file, err := updateCachedApk(pkg, device_codename)
	if err != nil {
		return -1, err
	}

	cmd := exec.Command("aapt", "dump", "badging", file)
	stdout, err := cmd.Output()
	if err != nil {
		return -1, err
	}

	re_vcode := regexp.MustCompile(`versionCode='(\d+)'`)
	matches := re_vcode.FindStringSubmatch(string(stdout))
	if len(matches) == 0 {
		return -1, fmt.Errorf("Failed to find %s's versionCode", pkg)
	}
	v, err := strconv.ParseInt(string(matches[1]), 10, 32)
	if err != nil {
		return -1, err
	}

	return int(v), nil
}

func getLatestVersionName(pkg string, device_codename string) (string, error) {
	file, err := updateCachedApk(pkg, device_codename)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("aapt", "dump", "badging", file)
	stdout, err := cmd.Output()
	if err != nil {
		return "", err
	}

	re_vname := regexp.MustCompile(`versionName='([^']+)'`)
	matches := re_vname.FindStringSubmatch(string(stdout))
	if len(matches) == 0 {
		return "", fmt.Errorf("Failed to find %s's versionName", pkg)
	}

	return string(matches[1]), nil
}

func pingDevice(serial string) error {
	cmd := exec.Command("adb", "connect", serial)
	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return fmt.Errorf("Failed to connect to %s", serial)
	}
	if !strings.Contains(string(stdouterr), fmt.Sprint("connected to ", serial)) {
		return fmt.Errorf("Device not connected: %s", stdouterr)
	}

	return nil
}

func installApk(serial string, pkg string, device_codename string) error {
	file, err := updateCachedApk(pkg, device_codename)
	if err != nil {
		return err
	}

	err = pingDevice(serial)
	if err != nil {
		return err
	}

	log.Println("Installing", pkg)
	cmd := exec.Command("adb", "-s", serial, "install", "-r", file)
	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return fmt.Errorf("Failed to install %s to %s: %s", pkg, serial, stdouterr)
	}
	if !strings.Contains(string(stdouterr), "Success") {
		return fmt.Errorf("Failed to install %s to %s: %s", pkg, serial, stdouterr)
	}

	return nil
}

func uninstallApk(serial string, pkg string) error {
	err := pingDevice(serial)
	if err != nil {
		return err
	}

	cmd := exec.Command("adb", "-s", serial, "uninstall", pkg)
	stdouterr, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	if !strings.Contains(string(stdouterr), "Success") {
		return fmt.Errorf("Failed to uninstall %s from %s", pkg, serial)
	}

	return nil
}

func resourceAndroidApkCreate(d *schema.ResourceData, m interface{}) error {
	device_codename := d.Get("device_codename").(string)
	pkg := d.Get("name").(string)
	serial := d.Get("adb_serial").(string)

	err := installApk(serial, pkg, device_codename)
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}

func resourceAndroidApkRead(d *schema.ResourceData, m interface{}) error {
	pkg := d.Get("name").(string)
	serial := d.Get("adb_serial").(string)

	err := pingDevice(serial)
	if err != nil {
		return err
	}

	cmd := exec.Command("adb", "-s", serial, "get-state")
	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to read state of %s", serial)
	}
	if string(stdout) != "device\n" {
		return fmt.Errorf("Device %s is not ready, in state: %s", serial, stdout)
	}

	cmd = exec.Command("adb", "-s", serial, "shell", "dumpsys", "package", pkg)
	stdout, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to read %s from %s", pkg, serial)
	}

	if !strings.Contains(string(stdout), fmt.Sprint("Unable to find package:", pkg)) {
		d.SetId("")
	}

	re_vcode := regexp.MustCompile(`versionCode=(\d+)`)
	matches := re_vcode.FindStringSubmatch(string(stdout))
	if len(matches) > 0 {
		v, err := strconv.ParseInt(string(matches[1]), 10, 32)
		if err != nil {
			return err
		}

		d.Set("version", v)
		d.SetId(fmt.Sprint(d.Get("adb_serial").(string), "-", pkg))
	} else {
		d.Set("version", -1)
		d.SetId("")
	}

	re_vname := regexp.MustCompile(`versionName=(.+) `)
	matches = re_vname.FindStringSubmatch(string(stdout))
	if len(matches) > 0 {
		d.Set("version_name", string(matches[1]))
	} else {
		d.Set("version_name", -1)
	}

	return nil
}

func resourceAndroidApkUpdate(d *schema.ResourceData, m interface{}) error {
	device_codename := d.Get("device_codename").(string)
	pkg := d.Get("name").(string)
	serial := d.Get("adb_serial").(string)

	err := installApk(serial, pkg, device_codename)
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}

func resourceAndroidApkDelete(d *schema.ResourceData, m interface{}) error {
	err := uninstallApk(d.Get("adb_serial").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}
