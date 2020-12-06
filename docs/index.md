---
layout: ""
page_title: "Provider: Android"
description: |-
  The Android provider is used to interact with Android devices, with resources representing state on the device itself.
---

# Android Provider

The Android provider is used to interact with Android devices, with resources representing state on the device itself. The device needs to be booted and connected (available to `adb`) before it can be used.

## Example Usage

```terraform
terraform {
  required_providers {
    android = {
      source = "OJFord/android"
    }
  }
}

provider "android" {
}
```

## Schema