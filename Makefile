.PHONY: help
help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

QUICWIRE_GCFLAGS?=
ECHO_PREFIX=@\#

dist:
	$(CMD_PREFIX) mkdir -p $@


##@ Build
.PHONY: build
build: dist ## Build quicwire
	$(ECHO_PREFIX) printf "  %-12s $@\n" "[GO BUILD]"
	$(CMD_PREFIX) CGO_ENABLED=0 go build -gcflags="$(QUICWIRE_GCFLAGS)" -o dist/qw ./cmd

.PHONY: build-stun
build-stun:  dist ## Build stun client
	$(ECHO_PREFIX) printf "  %-12s $@\n" "[GO BUILD]"
	$(CMD_PREFIX) CGO_ENABLED=0 go build -gcflags="$(QUICWIRE_GCFLAGS)" -o ./dist ./hack/stun-client

.PHONY: build-udpserver
build-udpserver:  dist ## Build udp server
	$(ECHO_PREFIX) printf "  %-12s $@\n" "[GO BUILD]"
	$(CMD_PREFIX) CGO_ENABLED=0 go build -gcflags="$(QUICWIRE_GCFLAGS)" -o ./dist ./hack/udpserver

.PHONY: build-udpclient
build-udpclient:  dist ## Build udp server
	$(ECHO_PREFIX) printf "  %-12s $@\n" "[GO BUILD]"
	$(CMD_PREFIX) CGO_ENABLED=0 go build -gcflags="$(QUICWIRE_GCFLAGS)" -o ./dist ./hack/udpclient

.PHONY: fire-stun
fire-stun:   ## Run stun client
	$(CMD_PREFIX) ./dist/stun-client -source-port 55380 -check-symmetric

.PHONY: prep
prep:  ## Format source code
	$(ECHO_PREFIX) printf "  %-12s $@\n" "[GO PREP]"
	$(CMD_PREFIX) CGO_ENABLED=0 go fmt ./...
	$(CMD_PREFIX) CGO_ENABLED=0 go vet ./...
	$(CMD_PREFIX) CGO_ENABLED=0 golint ./...

.PHONY: clean
clean: ## Clean quicwire binaries
	$(CMD_PREFIX) rm -rd dist

BINS := qw
OS := darwin windows linux linux
ARCH := amd64 amd64 amd64 arm64

# build qw for all target OS/Arch
.PHONY: build-all-os
build-all-os:
	@for bin in $(BINS); do \
		for os in $(OS); do \
			for arch in $(ARCH); do \
				output=dist/$$bin-$$os-$$arch; \
				[ $$os = "windows" ] && output=$$output.exe; \
				GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -gcflags="$(QUICWIRE_GCFLAGS)" -o $$output ./cmd/; \
			done \
		done \
	done
