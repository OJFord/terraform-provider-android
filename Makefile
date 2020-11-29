EXAMPLES := $(wildcard examples/*)

default: build

build: fmt
	go build -o terraform-provider-android

fmt:
	go fmt

examples: build $(EXAMPLES)
	for d in $(EXAMPLES); do \
		echo "Applying example $$d" ;\
		terraform init "$$d" ;\
		terraform apply -auto-approve "$$d" ;\
	done
