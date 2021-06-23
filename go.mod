module github.com/ory/x

replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2

replace github.com/dgrijalva/jwt-go => github.com/form3tech-oss/jwt-go v3.2.1+incompatible

replace github.com/oleiade/reflections => github.com/oleiade/reflections v1.0.1

require (
	github.com/bmatcuk/doublestar/v2 v2.0.3
	github.com/cockroachdb/cockroach-go/v2 v2.1.1
	github.com/dgraph-io/ristretto v0.0.3
	github.com/docker/docker v17.12.0-ce-rc1.0.20201201034508-7d75c1d40d88+incompatible
	github.com/fatih/structs v1.1.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/go-bindata/go-bindata v3.1.1+incompatible
	github.com/go-openapi/runtime v0.19.26
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gobuffalo/fizz v1.13.1-0.20201104174146-3416f0e6618f
	github.com/gobuffalo/httptest v1.0.2
	github.com/gobuffalo/packr v1.22.0
	github.com/gobuffalo/pop/v5 v5.3.3
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/mock v1.5.0
	github.com/google/go-jsonnet v0.17.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/inhies/go-bytesize v0.0.0-20201103132853-d0aed0d254f8
	github.com/instana/go-sensor v1.29.0
	github.com/jackc/pgconn v1.8.0
	github.com/jackc/pgx/v4 v4.10.1
	github.com/jandelgado/gcov2lcov v1.0.4
	github.com/jmoiron/sqlx v1.3.1
	github.com/julienschmidt/httprouter v1.3.0
	github.com/knadh/koanf v1.0.0
	github.com/lib/pq v1.10.0
	github.com/markbates/pkger v0.17.1
	github.com/opentracing/opentracing-go v1.2.0
	github.com/openzipkin-contrib/zipkin-go-opentracing v0.4.5
	github.com/openzipkin/zipkin-go v0.2.2
	github.com/ory/analytics-go/v4 v4.0.0
	github.com/ory/dockertest/v3 v3.6.5
	github.com/ory/go-acc v0.2.6
	github.com/ory/herodot v0.9.6
	github.com/ory/jsonschema/v3 v3.0.3
	github.com/pborman/uuid v1.2.1
	github.com/pelletier/go-toml v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/pkg/profile v1.2.1
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/common v0.15.0
	github.com/rs/cors v1.6.0
	github.com/rubenv/sql-migrate v0.0.0-20190212093014-1007f53448d7
	github.com/seatgeek/logrus-gelf-formatter v0.0.0-20210414080842-5b05eb8ff761
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cast v1.3.2-0.20200723214538-8d17101741c8
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/square/go-jose/v3 v3.0.0-20200630053402-0a67ce9b0693
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.7.1
	github.com/tidwall/sjson v1.1.5
	github.com/uber/jaeger-client-go v2.22.1+incompatible
	github.com/urfave/negroni v1.0.0
	go.elastic.co/apm v1.8.0
	go.elastic.co/apm/module/apmot v1.8.0
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.20.0
	go.opentelemetry.io/otel v0.20.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/mod v0.4.2
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	gonum.org/v1/plot v0.0.0-20200111075622-4abb28f724d5
	google.golang.org/grpc v1.36.0
	gopkg.in/DataDog/dd-trace-go.v1 v1.27.0
	gopkg.in/square/go-jose.v2 v2.5.1
)

go 1.16
