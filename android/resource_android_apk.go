package android

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/OJFord/terraform-provider-android/android/apk"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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
			"method": {
				Default:     "gplaycli",
				Description: "Method to use for acquiring the APK. (gplaycli, fdroid).",
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

func customiseDiff(d *schema.ResourceDiff, _ interface{}) error {
	device_codename := d.Get("device_codename").(string)
	apk, err := repo.Package(d.Get("method").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	v, err := repo.Version(apk, device_codename)
	if err != nil {
		return err
	}

	err = d.SetNew("version", v)
	if err != nil {
		return err
	}

	vold, vnew := d.GetChange("version")
	if vold.(int) > vnew.(int) {
		d.ForceNew("version")
	}

	vn, err := repo.VersionName(apk, device_codename)
	if err != nil {
		return err
	}

	err = d.SetNew("version_name", vn)
	if err != nil {
		return err
	}

	return nil
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

func installApk(serial string, apk repo.APKAcquirer, device_codename string) error {
	file, err := apk.UpdateCache(device_codename)
	if err != nil {
		return err
	}

	err = pingDevice(serial)
	if err != nil {
		return err
	}

	log.Println("Installing", apk.Name())
	cmd := exec.Command("adb", "-s", serial, "install", "-r", file)
	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return fmt.Errorf("Failed to install %s to %s: %s", apk.Name(), serial, stdouterr)
	}
	if !strings.Contains(string(stdouterr), "Success") {
		return fmt.Errorf("Failed to install %s to %s: %s", apk.Name(), serial, stdouterr)
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
	log.Println(string(stdouterr))
	if err != nil {
		return fmt.Errorf("Failed to uninstall %s from %s: %s", pkg, serial, stdouterr)
	}
	if !strings.Contains(string(stdouterr), "Success") {
		return fmt.Errorf("Failed to uninstall %s from %s", pkg, serial)
	}

	return nil
}

func resourceAndroidApkCreate(d *schema.ResourceData, m interface{}) error {
	device_codename := d.Get("device_codename").(string)
	serial := d.Get("adb_serial").(string)
	apk_acquirer, err := repo.Package(d.Get("method").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	err = installApk(serial, apk_acquirer, device_codename)
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
	serial := d.Get("adb_serial").(string)
	apk_acquirer, err := repo.Package(d.Get("method").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	err = installApk(serial, apk_acquirer, device_codename)
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
