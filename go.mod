module github.com/ory/x

replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2

replace github.com/dgrijalva/jwt-go => github.com/form3tech-oss/jwt-go v3.2.1+incompatible

replace github.com/oleiade/reflections => github.com/oleiade/reflections v1.0.1

replace github.com/gobuffalo/packr => github.com/gobuffalo/packr v1.30.1

replace github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.10

require (
	github.com/DataDog/datadog-go v4.8.2+incompatible // indirect
	github.com/DataDog/sketches-go v1.2.1 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/ajstarks/svgo v0.0.0-20210927141636-6d70534b1098 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/bmatcuk/doublestar/v2 v2.0.4
	github.com/bradleyjkemp/cupaloy/v2 v2.6.0
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cockroachdb/cockroach-go/v2 v2.2.1
	github.com/containerd/containerd v1.5.7 // indirect
	github.com/containerd/continuity v0.2.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/dgraph-io/ristretto v0.1.0
	github.com/docker/docker v20.10.9+incompatible
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/fatih/structs v1.1.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/ghodss/yaml v1.0.0
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-openapi/errors v0.20.1 // indirect
	github.com/go-openapi/runtime v0.20.0
	github.com/go-openapi/strfmt v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gobuffalo/fizz v1.14.0
	github.com/gobuffalo/here v0.6.2 // indirect
	github.com/gobuffalo/httptest v1.0.2
	github.com/gobuffalo/pop/v6 v6.0.1
	github.com/gofrs/uuid v4.1.0+incompatible
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/mock v1.6.0
	github.com/google/go-jsonnet v0.17.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.0
	github.com/inhies/go-bytesize v0.0.0-20210819104631-275770b98743
	github.com/instana/go-sensor v1.34.0
	github.com/jackc/pgconn v1.10.1
	github.com/jackc/pgx/v4 v4.13.0
	github.com/jandelgado/gcov2lcov v1.0.5
	github.com/jcchavezs/porto v0.3.0 // indirect
	github.com/jmoiron/sqlx v1.3.4
	github.com/julienschmidt/httprouter v1.3.0
	github.com/knadh/koanf v1.4.0
	github.com/lib/pq v1.10.4
	github.com/looplab/fsm v0.3.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/markbates/pkger v0.17.1
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/mitchellh/mapstructure v1.4.2 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/openzipkin-contrib/zipkin-go-opentracing v0.4.5
	github.com/openzipkin/zipkin-go v0.2.5
	github.com/ory/analytics-go/v4 v4.0.2
	github.com/ory/dockertest/v3 v3.8.1
	github.com/ory/go-acc v0.2.6
	github.com/ory/herodot v0.9.12
	github.com/ory/jsonschema/v3 v3.0.5
	github.com/pborman/uuid v1.2.1
	github.com/pelletier/go-toml v1.9.4
	github.com/pkg/errors v0.9.1
	github.com/pkg/profile v1.6.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.32.1
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rs/cors v1.8.0
	github.com/seatgeek/logrus-gelf-formatter v0.0.0-20210414080842-5b05eb8ff761
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cast v1.4.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/square/go-jose/v3 v3.0.0-20200630053402-0a67ce9b0693
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.9.4
	github.com/tidwall/sjson v1.2.2
	github.com/tinylib/msgp v1.1.6 // indirect
	github.com/uber/jaeger-client-go v2.29.1+incompatible
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/urfave/negroni v1.0.0
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	go.elastic.co/apm v1.14.0
	go.elastic.co/apm/module/apmot v1.14.0
	go.mongodb.org/mongo-driver v1.7.3 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.25.0
	go.opentelemetry.io/otel v1.2.0
	go.opentelemetry.io/otel/bridge/opentracing v1.2.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.2.0
	go.opentelemetry.io/otel/sdk v1.2.0
	go.opentelemetry.io/proto/otlp v0.10.0
	go.uber.org/atomic v1.9.0 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/mod v0.5.1
	golang.org/x/net v0.0.0-20211020060615-d418f374d309 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20211020174200-9d6173849985 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	gonum.org/v1/plot v0.10.0
	google.golang.org/genproto v0.0.0-20211020151524-b7c3a969101a // indirect
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/DataDog/dd-trace-go.v1 v1.33.0
	gopkg.in/ini.v1 v1.63.2 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0
	howett.net/plist v0.0.0-20201203080718-1454fab16a06 // indirect
)

go 1.16
