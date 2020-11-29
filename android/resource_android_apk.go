package android

import (
	"fmt"
	"github.com/adrg/xdg"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func resourceAndroidApk() *schema.Resource {
	return &schema.Resource{
		Create: resourceAndroidApkCreate,
		Read:   resourceAndroidApkRead,
		Update: resourceAndroidApkUpdate,
		Delete: resourceAndroidApkDelete,

		Schema: map[string]*schema.Schema{
			"installed": &schema.Schema{
				Computed: true,
				Type:     schema.TypeBool,
			},
			"installed_version": &schema.Schema{
				Computed: true,
				Type:     schema.TypeInt,
			},
			"installed_version_name": &schema.Schema{
				Computed: true,
				Type:     schema.TypeString,
			},
			"name": &schema.Schema{
				Required: true,
				Type:     schema.TypeString,
			},
			"target_version": &schema.Schema{
				Computed: true,
				Type:     schema.TypeInt,
			},
		},
	}
}

func updateCachedApk(pkg string) (string, error) {
	apk_dir, err := xdg.CacheFile("terraform-android/")
	if err != nil {
		return "", err
	}

	log.Println("Downloading", pkg)
	cmd := exec.Command("python", "-m", "gplaycli", fmt.Sprint("--folder=", apk_dir), fmt.Sprint("--download=", pkg))
	stdout, err := cmd.Output()
	log.Println(string(stdout))
	if err != nil {
		return "", err
	}
	log.Println(pkg, "downloaded")

	return fmt.Sprint(apk_dir, pkg, ".apk"), nil
}

func installApk(pkg string) error {
	file, err := updateCachedApk(pkg)
	if err != nil {
		return err
	}

	log.Println("Installing", pkg)
	cmd := exec.Command("adb", "install", "-r", file)
	stdout, err := cmd.Output()
	log.Println(string(stdout))
	if err != nil {
		log.Fatal(err)
		log.Fatal(string(err.(*exec.ExitError).Stderr))
		return err
	}
	if !strings.Contains(string(stdout), "Success") {
		return fmt.Errorf(string(stdout))
	}

	return nil
}

func uninstallApk(pkg string) error {
	cmd := exec.Command("adb", "uninstall", pkg)
	stdout, err := cmd.Output()
	if err != nil {
		return err
	}
	if !strings.Contains(string(stdout), "Success") {
		return fmt.Errorf(string(stdout))
	}

	return nil
}

func resourceAndroidApkCreate(d *schema.ResourceData, m interface{}) error {
	err := installApk(d.Get("name").(string))
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}

func resourceAndroidApkRead(d *schema.ResourceData, m interface{}) error {
	pkg := d.Get("name").(string)

	cmd := exec.Command("adb", "shell", "dumpsys", "package", pkg)
	stdout, err := cmd.Output()
	if err != nil {
		return err
	}

	d.Set("installed", !strings.Contains(string(stdout), fmt.Sprint("Unable to find package: ", pkg)))

	re_vcode := regexp.MustCompile(`versionCode=(\d+)`)
	matches := re_vcode.FindStringSubmatch(string(stdout))
	if len(matches) > 0 {
		d.Set("installed_version", matches[1])
	} else {
		d.Set("installed_version", -1)
	}

	re_vname := regexp.MustCompile(`versionName=([a-zA-Z0-9\.]+)`)
	matches = re_vname.FindStringSubmatch(string(stdout))
	if len(matches) > 0 {
		d.Set("installed_version_name", matches[1])
	} else {
		d.Set("installed_version_name", -1)
	}

	file, err := updateCachedApk(d.Get("name").(string))
	if err != nil {
		return err
	}

	cmd = exec.Command("aapt", "dump", "badging", file)
	stdout, err = cmd.Output()
	if err != nil {
		return err
	}

	re_vcode = regexp.MustCompile(`versionCode='(\d+)'`)
	matches = re_vcode.FindStringSubmatch(string(stdout))
	if len(matches) == 0 {
		return fmt.Errorf("Failed to find the acquired APK's versionCode")
	}
	d.Set("target_version", matches[1])

	return nil
}

func resourceAndroidApkUpdate(d *schema.ResourceData, m interface{}) error {
	err := installApk(d.Get("name").(string))
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}

func resourceAndroidApkDelete(d *schema.ResourceData, m interface{}) error {
	err := uninstallApk(d.Get("name").(string))
	if err != nil {
		return err
	}

	return resourceAndroidApkRead(d, m)
}
