.PHONY: init
init:
		GO111MODULE=on go install ./tools/listx github.com/ory/go-acc github.com/mattn/goveralls github.com/go-bindata/go-bindata/go-bindata github.com/golang/mock/mockgen

.PHONY: format
format:
		goreturns -w -i -local github.com/ory $$(listx . | grep -v "go_mod_indirect_pins.go")

.PHONY: test
test:
		make resetdb
		export TEST_DATABASE_POSTGRESQL=postgres://postgres:secret@127.0.0.1:3445/hydra?sslmode=disable; export TEST_DATABASE_COCKROACHDB=cockroach://root@127.0.0.1:3446/defaultdb?sslmode=disable; export TEST_DATABASE_MYSQL='mysql://root:secret@tcp(127.0.0.1:3444)/mysql?parseTime=true'; go test -race ./...

.PHONY: gen
gen:
		cd dbal; go-bindata -o migrate_files.go -pkg dbal ./stub/a ./stub/b ./stub/c ./stub/d

.PHONY: resetdb
resetdb:
		docker kill hydra_test_database_mysql || true
		docker kill hydra_test_database_postgres || true
		docker kill hydra_test_database_cockroach || true
		docker rm -f hydra_test_database_mysql || true
		docker rm -f hydra_test_database_postgres || true
		docker rm -f hydra_test_database_cockroach || true
		docker run --rm --name hydra_test_database_mysql -p 3444:3306 -e MYSQL_ROOT_PASSWORD=secret -d mysql:5.7
		docker run --rm --name hydra_test_database_postgres -p 3445:5432 -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=hydra -d postgres:9.6
		docker run --rm --name hydra_test_database_cockroach -p 3446:26257 -d cockroachdb/cockroach:v2.1.6 start --insecure

.PHONY: lint
lint:
		GO111MODULE=on golangci-lint run -v ./...
