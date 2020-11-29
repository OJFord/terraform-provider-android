package android

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"os"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"adb_serial": {
				Description: "Serial number (`get-serialno`) of the device - e.g. <IP>:<PORT> for TCP/IP devices",
				Required:    true,
				Type:        schema.TypeString,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"android_apk": resourceAndroidApk(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	err := os.Setenv("ANDROID_SERIAL", d.Get("adb_serial").(string))
	if err != nil {
		return nil, err
	}

	return nil, nil
}
