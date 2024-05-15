package types

type P2pConfigToml struct {
	Seeds               string `toml:"seeds"`
	Laddr               string `toml:"laddr"`
	PersistentPeers     string `toml:"persistent_peers"`
	MaxNumInboundPeers  int    `toml:"max_num_inbound_peers"`
	MaxNumOutboundPeers int    `toml:"max_num_outbound_peers"`
	SeedMode            bool   `toml:"seed_mode"`
}

type StateSyncConfigToml struct {
	Enable bool `toml:"enable"`
}

type ConsensusConfigToml struct {
	DoubleSignCheckHeight uint `toml:"double_sign_check_height"`
	SkipTimeoutCommit     bool `toml:"skip_timeout_commit"`
}

type TxIndexConfigToml struct {
	Indexer string `toml:"indexer"`
}

type ConfigToml struct {
	Moniker   string               `toml:"moniker"`
	P2P       *P2pConfigToml       `toml:"p2p"`
	StateSync *StateSyncConfigToml `toml:"statesync"`
	Consensus *ConsensusConfigToml `toml:"consensus"`
	TxIndex   *TxIndexConfigToml   `toml:"tx_index"`
}
