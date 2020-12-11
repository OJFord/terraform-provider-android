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
  endpoint = "${var.device_ip}:5555"
  name     = "com.fastmail.app"
}
