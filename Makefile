.PHONY: init
init:
		go get github.com/go-bindata/go-bindata/... github.com/mattn/goveralls github.com/ory/go-acc
		go install ./tools/listx

.PHONY: format
format:
		goreturns -w -i -local github.com/ory $$(listx .)

.PHONY: lint
lint:
		gometalinter --disable-all --enable=goimports --enable=gosec --enable=vet --enable=golint --deadline=3m --vendor ./...

.PHONY: test
test:
		go test -race ./...

.PHONY: gen
gen:
		cd dbal; go-bindata -o migrate_files.go -pkg dbal ./stub/a ./stub/b ./stub/c
