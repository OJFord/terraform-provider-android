resource "android_apk" "example" {
  adb_serial = "192.168.1.123"
  name       = "com.example.app"
}
