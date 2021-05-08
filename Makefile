EXAMPLES := $(wildcard examples/*)

default: build

build: fmt doc
	(cd android/apk/AuroraStore && ./gradlew assembleRelease)
	go build -o terraform-provider-android

fmt:
	gofmt -s -e -w .

doc:
	tfplugindocs

examples: build $(EXAMPLES)
	for d in $(EXAMPLES); do \
		echo "Applying example $$d" ;\
		terraform init "$$d" ;\
		terraform apply -auto-approve "$$d" ;\
	done
