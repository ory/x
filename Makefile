init:
		go get -u github.com/sqs/goreturns \
			github.com/go-bindata/go-bindata/...
		go install ./tools/listx

format:
		goreturns -w -i -local github.com/ory $$(listx .)

check:
		gometalinter --disable-all --enable=goimports --enable=gosec --enable=vet --enable=golint --vendor ./...

test:
		go test -race ./...

gen:
		cd dbal; go-bindata -o migrate_files.go -pkg dbal ./stub/a ./stub/b ./stub/c
