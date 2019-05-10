.PHONY: init
init:
		GO111MODULE=on go install ./tools/listx github.com/ory/go-acc github.com/mattn/goveralls github.com/go-bindata/go-bindata/go-bindata

.PHONY: format
format:
		goreturns -w -i -local github.com/ory $$(listx . | grep -v "mod_tools.go")

.PHONY: lint
lint:
		gometalinter --disable-all --enable=goimports --enable=gosec --enable=vet --enable=golint --deadline=3m --vendor ./...

.PHONY: test
test:
		go test -race ./...

.PHONY: gen
gen:
		cd dbal; go-bindata -o migrate_files.go -pkg dbal ./stub/a ./stub/b ./stub/c ./stub/d
