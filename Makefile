.PHONY: help
help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

NEXODUS_GCFLAGS?=
ECHO_PREFIX=@\#

dist:
	$(CMD_PREFIX) mkdir -p $@
	

##@ Build
.PHONY: build
build: dist ## Build quicnet
	$(ECHO_PREFIX) printf "  %-12s $@\n" "[GO BUILD]"
	$(CMD_PREFIX) CGO_ENABLED=0 go build -gcflags="$(NEXODUS_GCFLAGS)" -o dist/quicnet ./cmd

.PHONY: build-stun
build-stun:  dist ## Build stun client
	$(ECHO_PREFIX) printf "  %-12s $@\n" "[GO BUILD]"
	$(CMD_PREFIX) CGO_ENABLED=0 go build -gcflags="$(NEXODUS_GCFLAGS)" -o ./dist ./hack/stun-client

.PHONY: fire-stun
fire-stun:   ## Run stun client
	$(CMD_PREFIX) ./dist/stun-client -source-port 55380 -check-symmetric

.PHONY: format
format:  ## Format source code
	$(ECHO_PREFIX) printf "  %-12s $@\n" "[GO FMT]"
	$(CMD_PREFIX) CGO_ENABLED=0 go fmt ./...

.PHONY: clean
clean: dist ## Build quicnet
	$(CMD_PREFIX) rm -rd dist
