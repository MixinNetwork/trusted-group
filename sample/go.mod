module multisig

go 1.15

replace github.com/MixinNetwork/bot-api-go-client => ../../bot-api-go-client

require (
	cloud.google.com/go/logging v1.1.0
	github.com/MixinNetwork/bot-api-go-client v0.0.0-00010101000000-000000000000
	github.com/MixinNetwork/go-number v0.0.0-20180814121220-f48e2574d9ef
	github.com/MixinNetwork/mixin v0.9.0
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.0.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rubblelabs/ripple v0.0.0-20200627211644-1ecb0c494a6a
	github.com/stretchr/testify v1.4.0
	github.com/unrolled/render v1.0.3
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)
