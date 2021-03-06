package android

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"mvdan.cc/fdroidcl/adb"
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

type Device struct {
	*adb.Device
	endpoint string
	packages map[string]adb.Package
	serial   string
}

type Meta struct {
	devices map[string]Device
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	if !adb.IsServerRunning() {
		if err := adb.StartServer(); err != nil {
			return nil, err
		}
	}

	return Meta{
		make(map[string]Device),
	}, nil
}
