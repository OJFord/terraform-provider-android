.PHONY: clean-aurora-store clean-docs clean-provider default examples fmt
EXAMPLES := $(wildcard examples/*)
AURORASTORE := android/apk/AuroraStore/app/build/outputs/apk/debug/app-debug.apk

default: clean-docs clean-provider fmt terraform-provider-android docs

clean-aurora-store:
	rm $(AURORASTORE) || true

$(AURORASTORE):
	(cd android/apk/AuroraStore && ./gradlew assembleDebug)

clean-provider:
	rm terraform-provider-android || true

terraform-provider-android: $(AURORASTORE)
	go build -o terraform-provider-android

fmt:
	gofmt -s -e -w .

clean-docs:
	rm -r docs || true

docs: terraform-provider-android
	tfplugindocs

examples: build $(EXAMPLES)
	for d in $(EXAMPLES); do \
		echo "Applying example $$d" ;\
		terraform init "$$d" ;\
		terraform apply -auto-approve "$$d" ;\
	done
