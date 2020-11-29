terraform {
  required_providers {
    android = {
      source = "OJFord/android"
    }
  }
}

provider "android" {
}

resource "android_apk" "fastmail_example" {
  adb_serial = var.adb_serial
  name       = "com.fastmail.app"
}
