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
		Create: resourceAndroidApkCreate,
		Read:   resourceAndroidApkRead,
		Update: resourceAndroidApkUpdate,
		Delete: resourceAndroidApkDelete,

		Schema: map[string]*schema.Schema{
			"adb_serial": {
				Description: "Serial number (`get-serialno`) of the device - e.g. <IP>:<PORT> for TCP/IP devices",
				Required:    true,
				Type:        schema.TypeString,
			},
			"name": {
				ForceNew: true,
				Required: true,
				Type:     schema.TypeString,
			},
			"version": {
				Computed: true,
				Type:     schema.TypeInt,
			},
			"version_name": {
				Computed: true,
				Type:     schema.TypeString,
			},
		},

		CustomizeDiff: customiseDiff,
	}
}

func customiseDiff(diff *schema.ResourceDiff, _ interface{}) error {
	v, err := getLatestVersion(diff.Get("name").(string))
	if err != nil {
		return err
	}

	err = diff.SetNew("version", v)
	if err != nil {
		return err
	}

	vn, err := getLatestVersionName(diff.Get("name").(string))
	if err != nil {
		return err
	}

	err = diff.SetNew("version_name", vn)
	if err != nil {
		return err
	}

	return nil
}

func updateCachedApk(pkg string) (string, error) {
	apk_dir, err := xdg.CacheFile("terraform-android/")
	if err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	_, err = os.Stat(fmt.Sprint(apk_dir, "/", pkg, ".apk"))
	if os.IsNotExist(err) {
		log.Println("Downloading", pkg)
		cmd = exec.Command("python", "-m", "gplaycli", fmt.Sprint("--folder=", apk_dir), fmt.Sprint("--download=", pkg), "--device-codename=sailfish")
	} else {
		log.Println("Updating cached packages")
		cmd = exec.Command("python", "-m", "gplaycli", fmt.Sprint("--update=", apk_dir), "--yes", "--device-codename=sailfish")
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

func getLatestVersion(pkg string) (int, error) {
	file, err := updateCachedApk(pkg)
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

func getLatestVersionName(pkg string) (string, error) {
	file, err := updateCachedApk(pkg)
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

func installApk(serial string, pkg string) error {
	file, err := updateCachedApk(pkg)
	if err != nil {
		return err
	}

	log.Println("Installing", pkg)
	cmd := exec.Command("adb", "-s", serial, "install", "-r", file)
	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		log.Fatal(err)
		return err
	}
	if !strings.Contains(string(stdouterr), "Success") {
		return fmt.Errorf("Failed to install %s to %s", pkg, serial)
	}

	return nil
}

func uninstallApk(serial string, pkg string) error {
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
	err := installApk(d.Get("adb_serial").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}

func resourceAndroidApkRead(d *schema.ResourceData, m interface{}) error {
	pkg := d.Get("name").(string)
	serial := d.Get("adb_serial").(string)

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
	err := installApk(d.Get("adb_serial").(string), d.Get("name").(string))
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
