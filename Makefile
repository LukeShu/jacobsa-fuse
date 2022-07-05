goversion = 1.18

go-mod-tidy:
.PHONY: go-mod-tidy

go-mod-tidy: go-mod-tidy/main
go-mod-tidy/main:
	rm -f go.sum
	GOFLAGS=-mod=mod go mod tidy -go=$(goversion) -compat=$(goversion)
.PHONY: go-mod-tidy/main

go-mod-tidy: $(patsubst tools/src/%/go.mod,go-mod-tidy/tools/%,$(wildcard tools/src/*/go.mod))
go-mod-tidy/tools/%:
	rm -f tools/src/$*/go.sum
	cd tools/src/$* && GOFLAGS=-mod=mod go mod tidy -go=$(goversion) -compat=$(goversion)
.PHONY: go-mod-tidy/tools/%

tools/golangci-lint = tools/bin/golangci-lint
tools/bin/%: tools/src/%/pin.go tools/src/%/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)

lint: $(tools/golangci-lint)
	GOOS=linux   $(tools/golangci-lint) run ./...
	GOOS=darwin  $(tools/golangci-lint) run ./...
.PHONY: lint
