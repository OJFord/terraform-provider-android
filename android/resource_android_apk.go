package android

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/OJFord/terraform-provider-android/android/apk"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"mvdan.cc/fdroidcl/adb"
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

func getDevice(serial string) (*adb.Device, error) {
	cmd := exec.Command("adb", "connect", serial)
	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %s", serial)
	}
	if !strings.Contains(string(stdouterr), fmt.Sprint("connected to ", serial)) {
		return nil, fmt.Errorf("Device not connected: %s", stdouterr)
	}

	devices, err := adb.Devices()
	if err != nil {
		return nil, fmt.Errorf("Failed to get devices: %s", err)
	}

	for _, device := range devices {
		log.Println("Found device", device.ID)
		if device.ID == serial {
			return device, nil
		}
	}

	return nil, fmt.Errorf("Could not find %s", serial)
}

func installApk(serial string, apk repo.APKAcquirer, device_codename string) error {
	file, err := apk.UpdateCache(device_codename)
	if err != nil {
		return err
	}

	device, err := getDevice(serial)
	if err != nil {
		return err
	}

	log.Println("Installing", apk.Name())
	if err = device.Install(file); err != nil {
		return fmt.Errorf("Failed to install %s to %s: %s", apk.Name(), device.Model, err)
	}

	return nil
}

func uninstallApk(serial string, pkg string) error {
	device, err := getDevice(serial)
	if err != nil {
		return err
	}

	if err = device.Uninstall(pkg); err != nil {
		return fmt.Errorf("Failed to uninstall %s from %s: %s", pkg, device.Model, err)
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

	device, err := getDevice(serial)
	if err != nil {
		return err
	}

	cmd := device.AdbCmd("get-state")
	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to read state of %s", serial)
	}
	if string(stdout) != "device\n" {
		return fmt.Errorf("Device %s is not ready, in state: %s", serial, stdout)
	}

	installed, err := device.Installed()
	if err != nil {
		return fmt.Errorf("Failed to read %s's packages", device.Model)
	}

	var pkg_info *adb.Package
	for installed_pkg, installed_pkg_info := range installed {
		log.Printf("%s has %s", device.Model, installed_pkg)
		if installed_pkg == pkg {
			pkg_info = &installed_pkg_info
		}
	}

	if pkg_info == nil {
		d.SetId("")
		d.Set("version", -1)
		d.Set("version_name", "")
	} else {
		d.SetId(fmt.Sprint(d.Get("adb_serial").(string), "-", pkg))
		d.Set("version", pkg_info.VersCode)
		d.Set("version_name", pkg_info.VersName)
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
