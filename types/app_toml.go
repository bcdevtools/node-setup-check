package types

type ApiAppToml struct {
	Enable  bool `toml:"enable"`
	Swagger bool `toml:"swagger"`
}

type JsonRpcAppToml struct {
	Enable        bool `toml:"enable"`
	EnableIndexer bool `toml:"enable-indexer"`
}

type StateSyncAppToml struct {
	SnapshotInterval   uint `toml:"snapshot-interval"`
	SnapshotKeepRecent uint `toml:"snapshot-keep-recent"`
}

type GrpcAppToml struct {
	Enable         bool   `toml:"enable"`
	Address        string `toml:"address"`
	MaxSendMsgSize string `toml:"max-send-msg-size"`
}

type AppToml struct {
	MinimumGasPrices  string            `toml:"minimum-gas-prices"`
	Pruning           string            `toml:"pruning"`
	PruningKeepRecent string            `toml:"pruning-keep-recent"`
	PruningInterval   string            `toml:"pruning-interval"`
	HaltHeight        int64             `toml:"halt-height"`
	HaltTime          int64             `toml:"halt-time"`
	MinRetainsBlock   uint              `toml:"min-retain-blocks"`
	Api               *ApiAppToml       `toml:"api"`
	JsonRpc           *JsonRpcAppToml   `toml:"json-rpc"`
	StateSync         *StateSyncAppToml `toml:"state-sync"`
	Grpc              *GrpcAppToml      `toml:"grpc"`
}
