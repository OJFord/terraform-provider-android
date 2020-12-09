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
			"endpoint": {
				Description: "IP:PORT of the device. Required for ADB over WiFi, omit for USB connections.",
				Optional:    true,
				Computed:    true,
				AtLeastOneOf: []string{
					"endpoint",
					"serial",
				},
				Type: schema.TypeString,
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
			"serial": {
				Description: "Serial number (`getprop ro.serialno`) of the device.",
				ForceNew:    true,
				Optional:    true,
				Computed:    true,
				AtLeastOneOf: []string{
					"endpoint",
					"serial",
				},
				Type: schema.TypeString,
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
	apk, err := repo.Package(d.Get("method").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	serial, endpoint := d.Get("serial").(string), d.Get("endpoint").(string)

	device, serial, err := findDeviceBySerialOrEndpoint(serial, endpoint)
	if err != nil {
		log.Println("Failed to find device, not updating cached APK")
	} else {
		log.Printf("Found %s ('%s') @ %s", serial, device.Device, device.ID)

		if err = d.SetNew("endpoint", device.ID); err != nil {
			return err
		}

		if err = d.SetNew("serial", serial); err != nil {
			return err
		}

		if _, err = apk.UpdateCache(device); err != nil {
			return err
		}
	}

	v, err := repo.Version(apk)
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

	vn, err := repo.VersionName(apk)
	if err != nil {
		return err
	}

	err = d.SetNew("version_name", vn)
	if err != nil {
		return err
	}

	return nil
}

func connectDevice(endpoint string) (*adb.Device, error) {
	log.Println("Finding device", endpoint)

	cmd := exec.Command("adb", "connect", endpoint)
	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))

	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %s", endpoint)
	}
	if !strings.Contains(string(stdouterr), fmt.Sprint("connected to ", endpoint)) {
		return nil, fmt.Errorf("Device not connected: %s", stdouterr)
	}

	devices, err := adb.Devices()
	if err != nil {
		return nil, fmt.Errorf("Failed to get devices: %s", err)
	}

	for _, device := range devices {
		log.Println("Found device", device.ID)
		if device.ID == endpoint {
			return device, nil
		}
	}

	return nil, fmt.Errorf("Could not find %s", endpoint)
}

func getDevice(serial string) (*adb.Device, error) {
	devices, err := adb.Devices()
	if err != nil {
		return nil, fmt.Errorf("Failed to get devices: %s", err)
	}

	for _, device := range devices {
		log.Println("Found device", device.ID)
		if props, err := device.AdbProps(); props["ro.serialno"] == serial {
			return device, nil
		} else {
			log.Println("[ERROR]", err)
		}
	}

	return nil, fmt.Errorf("Could not find %s", serial)
}

func findDeviceBySerialOrEndpoint(serial string, endpoint string) (*adb.Device, string, error) {
	log.Printf("Looking for device %s at %s", serial, endpoint)

	if endpoint != "" {
		device, err := connectDevice(endpoint)
		if err != nil {
			return nil, serial, err
		}

		props, err := device.AdbProps()
		if err != nil {
			return nil, serial, err
		}

		epSerial := props["ro.serialno"]
		log.Printf("[INFO] %s is %s", endpoint, epSerial)
		if serial != "" && epSerial != serial {
			return nil, serial, fmt.Errorf("Device found at %s is %s, not %s.", endpoint, epSerial, serial)
		}

		return device, epSerial, nil
	}

	if serial != "" {
		device, err := getDevice(serial)
		if err != nil {
			return nil, serial, err
		}

		return device, serial, err
	}

	return nil, "", fmt.Errorf("No endpoint or serial specified")
}

func installApk(device *adb.Device, apk repo.APKAcquirer) error {
	log.Println("Installing", apk.Name())
	if err := device.Install(repo.Path(apk)); err != nil {
		return fmt.Errorf("Failed to install %s to %s: %s", apk.Name(), device.Model, err)
	}

	return nil
}

func uninstallApk(device *adb.Device, pkg string) error {
	if err := device.Uninstall(pkg); err != nil {
		return fmt.Errorf("Failed to uninstall %s from %s: %s", pkg, device.Model, err)
	}

	return nil
}

func resourceAndroidApkCreate(d *schema.ResourceData, m interface{}) error {
	serial, endpoint := d.Get("serial").(string), d.Get("endpoint").(string)

	device, _, err := findDeviceBySerialOrEndpoint(serial, endpoint)
	if err != nil {
		return err
	}

	apk_acquirer, err := repo.Package(d.Get("method").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	err = installApk(device, apk_acquirer)
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}

func resourceAndroidApkRead(d *schema.ResourceData, m interface{}) error {
	serial, endpoint := d.Get("serial").(string), d.Get("endpoint").(string)

	device, serial, err := findDeviceBySerialOrEndpoint(serial, endpoint)
	if err != nil {
		return err
	}

	d.Set("serial", serial)

	cmd := device.AdbCmd("get-state")
	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to read state of %s", serial)
	}
	if string(stdout) != "device\n" {
		return fmt.Errorf("Device %s is not ready, in state: %s", serial, stdout)
	}

	// Seems to be an upstream bug here, they report ABI versions or other incorrect numbers
	/*
			installed, err := device.Installed()
			if err != nil {
				return fmt.Errorf("Failed to read %s's packages", device.Model)
			}

			pkg := d.Get("name").(string)
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
				d.SetId(fmt.Sprint(serial, "-", pkg))

				d.Set("version", pkg_info.VersCode)
				d.Set("version_name", pkg_info.VersName)
		    }
	*/

	pkg := d.Get("name").(string)
	cmd = device.AdbShell("dumpsys", "package", pkg)
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

		d.SetId(fmt.Sprint(serial, "-", pkg))
		d.Set("version", v)
	} else {
		d.SetId("")
		d.Set("version", -1)
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
	serial, endpoint := d.Get("serial").(string), d.Get("endpoint").(string)

	device, _, err := findDeviceBySerialOrEndpoint(serial, endpoint)
	if err != nil {
		return err
	}

	apk_acquirer, err := repo.Package(d.Get("method").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	err = installApk(device, apk_acquirer)
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}

func resourceAndroidApkDelete(d *schema.ResourceData, m interface{}) error {
	serial, endpoint := d.Get("serial").(string), d.Get("endpoint").(string)

	device, _, err := findDeviceBySerialOrEndpoint(serial, endpoint)
	if err != nil {
		return err
	}

	err = uninstallApk(device, d.Get("name").(string))
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}
