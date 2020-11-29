terraform {
  required_providers {
    android = {
      source = "OJFord/android"
    }
  }
}

provider "android" {
  adb_serial = "192.168.1.217:5555"
}

resource "android_apk" "fastmail_example" {
  name = "com.fastmail.app"
}
