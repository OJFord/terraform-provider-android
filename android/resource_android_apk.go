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

func customiseDiff(d *schema.ResourceDiff, m interface{}) error {
	apk, err := repo.Package(d.Get("method").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	serial, endpoint := d.Get("serial").(string), d.Get("endpoint").(string)

	device, err := findDeviceBySerialOrEndpoint(serial, endpoint, m.(Meta))
	if err != nil {
		return err
	}

	if err = d.SetNew("endpoint", device.ID); err != nil {
		return err
	}

	if err = d.SetNew("serial", serial); err != nil {
		return err
	}

	if _, err = apk.UpdateCache(device.Device); err != nil {
		return err
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

	var found []string = make([]string, 0)
	for _, device := range devices {
		log.Println("Found device", device.ID)
		if device.ID == endpoint {
			return device, nil
		}
		found = append(found, device.ID)
	}

	return nil, fmt.Errorf("Could not find %s - perhaps you meant one of %s?", endpoint, found)
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

func getCachedDevice(endpoint string, m Meta) *Device {
	if device, ok := m.devices[endpoint]; ok && device.Device != nil {
		log.Printf("Found cached device %s: %s", endpoint, device.serial)
		return &device
	}

	log.Println(endpoint, "not found in cache:", m.devices)
	return nil
}

func findDeviceBySerialOrEndpoint(serial string, endpoint string, m Meta) (*Device, error) {
	log.Printf("Looking for device %s at %s", serial, endpoint)

	if endpoint != "" {
		if device := getCachedDevice(endpoint, m); device != nil {
			return device, nil
		}

		device := Device{}

		var err error
		device.Device, err = connectDevice(endpoint)
		if err != nil {
			return nil, err
		}

		props, err := device.AdbProps()
		if err != nil {
			return nil, err
		}

		device.serial = props["ro.serialno"]
		log.Printf("[INFO] %s is %s", endpoint, device.serial)
		if serial != "" && device.serial != serial {
			return nil, fmt.Errorf("Device found at %s is %s, not %s.", endpoint, device.serial, serial)
		}

		m.devices[endpoint] = device
		return &device, nil
	}

	if serial != "" {
		endpoint := fmt.Sprintf("USB-%s", serial)
		if device := getCachedDevice(endpoint, m); device != nil {
			return device, nil
		}

		device := Device{}
		var err error
		device.Device, err = getDevice(serial)

		m.devices[endpoint] = device
		return &device, err
	}

	return nil, fmt.Errorf("No endpoint or serial specified")
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

	device, err := findDeviceBySerialOrEndpoint(serial, endpoint, m.(Meta))
	if err != nil {
		return err
	}

	apk_acquirer, err := repo.Package(d.Get("method").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	err = installApk(device.Device, apk_acquirer)
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}

func resourceAndroidApkRead(d *schema.ResourceData, m interface{}) error {
	serial, endpoint := d.Get("serial").(string), d.Get("endpoint").(string)

	device, err := findDeviceBySerialOrEndpoint(serial, endpoint, m.(Meta))
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

	var installed map[string]adb.Package
	if len(device.packages) == 0 {
		var err error
		if installed, err = device.Installed(); err != nil {
			return fmt.Errorf("Failed to read packages from %s: %s", serial, err)
		}

		device.packages = installed
	}

	pkg := d.Get("name").(string)
	if ipkg, ok := installed[pkg]; ok {
		d.SetId(fmt.Sprint(serial, "-", pkg))
		d.Set("version", ipkg.VersCode)
		d.Set("version_name", ipkg.VersName)
		return nil
	}

	d.SetId("")
	d.Set("version", -1)
	d.Set("version_name", "Not installed")
	return nil
}

func resourceAndroidApkUpdate(d *schema.ResourceData, m interface{}) error {
	serial, endpoint := d.Get("serial").(string), d.Get("endpoint").(string)

	device, err := findDeviceBySerialOrEndpoint(serial, endpoint, m.(Meta))
	if err != nil {
		return err
	}

	apk_acquirer, err := repo.Package(d.Get("method").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	err = installApk(device.Device, apk_acquirer)
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}

func resourceAndroidApkDelete(d *schema.ResourceData, m interface{}) error {
	serial, endpoint := d.Get("serial").(string), d.Get("endpoint").(string)

	device, err := findDeviceBySerialOrEndpoint(serial, endpoint, m.(Meta))
	if err != nil {
		return err
	}

	err = uninstallApk(device.Device, d.Get("name").(string))
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}
