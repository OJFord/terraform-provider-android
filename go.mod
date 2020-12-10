module github.com/OJFord/terraform-provider-android

go 1.15

replace mvdan.cc/fdroidcl => github.com/OJFord/fdroidcl v0.5.1-0.20201210224942-bb8350dd7167

require (
	github.com/adrg/xdg v0.2.3
	github.com/hashicorp/terraform-plugin-sdk v1.16.0
	mvdan.cc/fdroidcl v0.5.0
)
