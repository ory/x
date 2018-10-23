init:
		go get -u github.com/sqs/goreturns

format:
		goreturns -w -i -local github.com/ory .

test:
		go test -race ./...
