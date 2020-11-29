# Android provider for [Terraform](https://terraform.io)

For usage, see [the docs](https://registry.terraform.io/providers/OJFord/android/latest/docs).

## System dependencies

### `resource "android_apk"`

Currently depends on (on the machine running terraform) the following binaries in `$PATH`:
- `aapt` (from android-sdk-build-tools)
- `adb` (from android-tools)
- `gplaycli` (from python-pip: gplaycli)

It is desirable to reduce/eliminate these and use Go libraries instead. (PRs very welcome!)
