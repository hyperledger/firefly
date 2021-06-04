module github.com/kaleido-io/firefly

go 1.16

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/Masterminds/squirrel v1.5.0
	github.com/aidarkhanov/nanoid v1.0.8
	github.com/akamensky/base58 v0.0.0-20170920141933-92b0f56f531a
	github.com/bombsimon/wsl/v2 v2.0.0 // indirect
	github.com/briandowns/spinner v1.15.0 // indirect
	github.com/coocood/freecache v1.1.1
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/docker/go-units v0.4.0
	github.com/fatih/color v1.12.0 // indirect
	github.com/getkin/kin-openapi v0.62.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-lintpack/lintpack v0.5.2 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-resty/resty/v2 v2.6.0
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/golang-migrate/migrate v3.5.4+incompatible // indirect
	github.com/golang-migrate/migrate/v4 v4.14.2-0.20210521165626-8a1a8534dc64
	github.com/golangci/errcheck v0.0.0-20181223084120-ef45e06d44b6 // indirect
	github.com/golangci/goconst v0.0.0-20180610141641-041c5f2b40f3 // indirect
	github.com/golangci/gocyclo v0.0.0-20180528134321-2becd97e67ee // indirect
	github.com/golangci/golangci-lint v1.40.1 // indirect
	github.com/golangci/ineffassign v0.0.0-20190609212857-42439a7714cc // indirect
	github.com/golangci/prealloc v0.0.0-20180630174525-215b22d4de21 // indirect
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4
	github.com/jarcoal/httpmock v1.0.8
	github.com/kaleido-io/firefly-cli v0.0.8 // indirect
	github.com/karlseguin/ccache v2.0.3+incompatible
	github.com/klauspost/cpuid v1.2.0 // indirect
	github.com/klauspost/cpuid/v2 v2.0.6 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.10.2
	github.com/likexian/gokit v0.24.7
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-sqlite3 v1.14.6 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/microcosm-cc/bluemonday v1.0.9
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/multiformats/go-multiaddr v0.3.2 // indirect
	github.com/multiformats/go-multihash v0.0.15 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.15.0 // indirect
	github.com/onsi/gomega v1.10.5 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pelletier/go-toml v1.9.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.10.0 // indirect
	github.com/rs/cors v1.7.0
	github.com/securego/gosec v0.0.0-20191002120514-e680875ea14d // indirect
	github.com/shirou/gopsutil v0.0.0-20190901111213-e4ec7b275ada // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/snowflakedb/glog v0.0.0-20180824191149-f5055e6f21ce // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/tommy-muehle/go-mnd v1.1.1 // indirect
	github.com/ugorji/go v1.1.4 // indirect
	github.com/vektra/mockery v1.1.2 // indirect
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	golang.org/x/net v0.0.0-20210521195947-fe42d452be8f // indirect
	golang.org/x/sys v0.0.0-20210603125802-9665404d3644 // indirect
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56 // indirect
	golang.org/x/text v0.3.6
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	gotest.tools v2.2.0+incompatible
	modernc.org/b v1.0.1 // indirect
	modernc.org/db v1.0.1 // indirect
	modernc.org/file v1.0.2 // indirect
	modernc.org/golex v1.0.1 // indirect
	modernc.org/lldb v1.0.1 // indirect
	modernc.org/mathutil v1.2.2 // indirect
	modernc.org/ql v1.3.1
	modernc.org/sqlite v1.10.7
	modernc.org/strutil v1.1.1 // indirect
	modernc.org/zappy v1.0.3 // indirect
	sourcegraph.com/sqs/pbtypes v0.0.0-20180604144634-d3ebe8f20ae4 // indirect
)
