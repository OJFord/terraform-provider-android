---
page_title: "Resource android_apk - terraform-provider-android"
subcategory: ""
description: |-
  Provides an Android APK resource. This can be used to create, read, update, and delete installed APKs (apps) on your Android device.
---

# android_apk `Resource`

Provides an Android APK resource. This can be used to create, read, update, and delete installed APKs (apps) on your Android device.

## System dependencies

Currently CRUDing an `android_apk` resource depends on the following binaries in `$PATH`:
- `aapt` (from android-sdk-build-tools)
- `adb` (from android-tools)
- `gplaycli` (from python-pip: gplaycli)

It is intended to reduce/eliminate these (cf. [GitHub#4](//github.com/OJFord/terraform-provider-android/issues/4)), but for now, they're required (on the machine running `terraform`).

## Example Usage

```terraform
resource "android_apk" "example" {
  adb_serial = "192.168.1.123"
  name       = "com.example.app"
}
```


## Schema

### Required

- **adb_serial** (String, Required) Serial number (`get-serialno`) of the device - e.g. <IP>:<PORT> for TCP/IP devices
- **name** (String, Required) Qualified name of the package to install, e.g. `com.google.zxing.client.android`

### Optional

- **id** (String, Optional) The ID of this resource.
- **method** (String, Optional) Method to use for acquiring the APK. (gplaycli, fdroid).

### Read-only

- **version** (Number, Read-only) Monotonically increasing `versionCode` of the package, safe for comparison
- **version_name** (String, Read-only) Human-friendly `versionName`, defined by the package author and not guaranteed to increment


