goversion = 1.18

go-mod-tidy:
.PHONY: go-mod-tidy

go-mod-tidy: go-mod-tidy/main
go-mod-tidy/main:
	rm -f go.sum
	GOFLAGS=-mod=mod go mod tidy -go=$(goversion) -compat=$(goversion)
.PHONY: go-mod-tidy/main
