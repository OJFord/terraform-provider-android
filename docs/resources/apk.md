---
page_title: "android_apk Resource - terraform-provider-android"
subcategory: ""
description: |-
  
---

# Resource `android_apk`





## Schema

### Required

- **adb_serial** (String, Required) Serial number (`get-serialno`) of the device - e.g. <IP>:<PORT> for TCP/IP devices
- **name** (String, Required) Qualified name of the package to install, e.g. `com.google.zxing.client.android`

### Optional

- **device_codename** (String, Optional) Device codename to present to Play Store (may affect app availability, or more?)
- **id** (String, Optional) The ID of this resource.

### Read-only

- **version** (Number, Read-only) Monotonically increasing `versionCode` of the package, safe for comparison
- **version_name** (String, Read-only) Human-friendly `versionName`, defined by the package author and not guaranteed to increment


