init:
		go get -u github.com/sqs/goreturns
		go install ./tools/listx

format:
		goreturns -w -i -local github.com/ory $$(listx .)

test:
		go test -race ./...
