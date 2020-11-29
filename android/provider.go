package android

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"os/exec"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},
		ResourcesMap: map[string]*schema.Resource{
			"android_apk": resourceAndroidApk(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	cmd := exec.Command("adb", "start-server")
	_, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return nil, nil
}
