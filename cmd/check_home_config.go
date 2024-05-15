package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/bcdevtools/node-setup-check/constants"
	"github.com/bcdevtools/node-setup-check/types"
	"github.com/bcdevtools/node-setup-check/utils"
	"github.com/pelletier/go-toml/v2"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

func checkHomeConfig(home string, nodeType types.NodeType) {
	configPath := path.Join(home, "config")
	perm, exists, isDir, err := utils.FileInfo(configPath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check config directory at %s: %v\n", configPath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: config directory does not exist: %s\n", configPath)
		return
	}
	if !isDir {
		exitWithErrorMsgf("ERR: config is not a directory: %s\n", configPath)
		return
	}

	filePerm := types.FilePermFrom(perm)
	if filePerm.Other.Write {
		fatalRecord("config directory is writable by others", "chmod o-w "+configPath)
	}
	if filePerm.Group.Write {
		fatalRecord("config directory is writable by group", "chmod g-w "+configPath)
	}
	if !filePerm.User.IsFullPermission() {
		fatalRecord("config directory is not fully accessible by user", "chmod u+rwx "+configPath)
	}

	appToml := checkHomeConfigAppToml(configPath, nodeType)
	checkHomeConfigClientToml(configPath)
	configToml := checkHomeConfigConfigToml(configPath, nodeType)
	checkHomeConfigGenesisJson(configPath)
	checkHomeConfigNodeKeyJson(configPath)
	checkHomeConfigPrivValidatorKeyJson(configPath)
	checkHomeConfigConfigTomlAndAppToml(nodeType, configToml, appToml)
}

func checkHomeConfigAppToml(configPath string, nodeType types.NodeType) *types.AppToml {
	isValidator := nodeType == types.ValidatorNode
	isRpc := nodeType == types.RpcNode
	isSnapshotNode := nodeType == types.SnapshotNode
	isArchivalNode := nodeType == types.ArchivalNode
	appTomlFilePath := path.Join(configPath, "app.toml")
	perm, exists, isDir, err := utils.FileInfo(appTomlFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check app.toml file at %s: %v\n", appTomlFilePath, err)
		return nil
	}
	if !exists {
		exitWithErrorMsgf("ERR: app.toml file does not exist: %s\n", appTomlFilePath)
		return nil
	}
	if isDir {
		exitWithErrorMsgf("ERR: app.toml is a directory, it should be a file: %s\n", appTomlFilePath)
		return nil
	}
	filePerm := types.FilePermFrom(perm)
	if filePerm.Other.Write {
		fatalRecord("app.toml file is writable by others", "chmod 644 "+appTomlFilePath)
	}
	if filePerm.Group.Write {
		fatalRecord("app.toml file is writable by group", "chmod 644 "+appTomlFilePath)
	}
	if !filePerm.User.Read {
		fatalRecord("app.toml file is not readable by user", "chmod 644 "+appTomlFilePath)
	}
	if !filePerm.User.Write {
		fatalRecord("app.toml file is not writable by user", "chmod 644 "+appTomlFilePath)
	}

	bz, err := os.ReadFile(appTomlFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to read app.toml file at %s: %v\n", appTomlFilePath, err)
		return nil
	}

	var app types.AppToml
	err = toml.Unmarshal(bz, &app)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to unmarshal app.toml file at %s: %v\n", appTomlFilePath, err)
		return nil
	}

	if app.MinimumGasPrices == "" {
		if isValidator {
			warnRecord("minimum-gas-prices is empty, validator must set, in app.toml file", "")
		} else {
			warnRecord("minimum-gas-prices is empty in app.toml file", "")
		}
	} else if regexp.MustCompile(`^\s*0[a-z]+\s*$`).MatchString(app.MinimumGasPrices) {
		if isValidator {
			warnRecord(fmt.Sprintf("minimum-gas-prices is zero, validator must set, in app.toml file: %s", app.MinimumGasPrices), "")
		} else {
			warnRecord(fmt.Sprintf("minimum-gas-prices is zero in app.toml file: %s", app.MinimumGasPrices), "")
		}
	}

	switch app.Pruning {
	case constants.PruningDefault:
		if isValidator {
			warnRecord(
				"pruning set to 'default' in app.toml file",
				"set pruning to everything for validator",
			)
		} else if isSnapshotNode {
			warnRecord(
				"pruning set to 'default' in app.toml file, snapshot not should be configured properly for snapshot purpose",
				"set pruning to custom 100/10",
			)
		} else if isArchivalNode {
			fatalRecord(
				"pruning set to 'default' in app.toml file, archival node should be configured properly for archival purpose",
				"set pruning to nothing",
			)
		}
	case constants.PruningNothing:
		if isValidator {
			fatalRecord(
				"pruning set to 'nothing' in app.toml file, validator should not use this option",
				"set pruning to everything",
			)
		} else if isSnapshotNode {
			fatalRecord(
				"pruning set to 'nothing' in app.toml file, snapshot not should be configured properly for snapshot purpose",
				"set pruning to custom 100/10",
			)
		}
	case constants.PruningEverything:
		if isValidator {
			//
		} else if isArchivalNode {
			fatalRecord(
				"pruning set to 'everything' in app.toml file, archival node must not use this option",
				"set pruning to nothing",
			)
		} else {
			fatalRecord(
				"pruning set to 'everything' in app.toml file, non-validator should not use this option",
				"set pruning to default or custom",
			)
		}
	case constants.PruningCustom:
		if isArchivalNode {
			fatalRecord("pruning set to 'custom' in app.toml file, archival node must not use this option", "set pruning to nothing")
		}
	default:
		msg := fmt.Sprintf("invalid pruning option '%s' in app.toml file", app.Pruning)
		if isValidator {
			fatalRecord(msg, "set pruning to everything")
		} else if isArchivalNode {
			fatalRecord(msg, "set pruning to nothing")
		} else {
			fatalRecord(msg, "set pruning to default or custom")
		}
		exitWithErrorMsgf("ERR: invalid pruning option in app.toml file: %s\n", appTomlFilePath, app.Pruning)
		return nil
	}

	if isSnapshotNode {
		if app.Pruning != constants.PruningCustom || app.PruningKeepRecent != "100" || app.PruningInterval != "10" {
			warnRecord("snapshot node should use pruning custom 100/10 in app.toml file", "set pruning to custom 100/10")
		}
	}

	if app.Pruning == constants.PruningCustom {
		if app.PruningKeepRecent != "" {
			pruningKeepRecent, err := strconv.ParseInt(app.PruningKeepRecent, 10, 64)
			if err != nil {
				exitWithErrorMsgf("ERR: failed to parse pruning-keep-recent in app.toml file: %v\n", err)
				return nil
			}

			if pruningKeepRecent > 400_000 {
				warnRecord("pruning-keep-recent is too high in app.toml file", "")
			} else if pruningKeepRecent < 2 {
				fatalRecord("pruning-keep-recent is too low in app.toml file", "")
			}
		} else {
			fatalRecord("pruning-keep-recent is empty in app.toml file", "set pruning-keep-recent to 100")
		}

		if app.PruningInterval != "" {
			pruningInterval, err := strconv.ParseInt(app.PruningInterval, 10, 64)
			if err != nil {
				exitWithErrorMsgf("ERR: failed to parse pruning-interval in app.toml file: %v\n", err)
				return nil
			}

			if pruningInterval > 10_000 {
				warnRecord("pruning-interval is too high in app.toml file", "")
			} else if pruningInterval < 10 {
				fatalRecord("pruning-interval is too low in app.toml file", "")
			}
		} else {
			fatalRecord("pruning-interval is empty in app.toml file", "set pruning-interval to 10")
		}
	}

	if app.HaltHeight > 0 {
		warnRecord(fmt.Sprintf("halt-height is set to %d in app.toml file", app.HaltHeight), "unset halt-height unless on purpose")
	}

	if app.HaltTime > 0 {
		warnRecord(fmt.Sprintf("halt-time is set to %d in app.toml file", app.HaltTime), "unset halt-time unless on purpose")
	}

	if app.Pruning == constants.PruningDefault {
		if app.MinRetainsBlock < 362880 {
			warnRecord(
				"min-retain-blocks should be set to 362880 if pruning \"default\" in app.toml file",
				"set min-retain-blocks to 362880",
			)
		}
	} else if app.Pruning == constants.PruningEverything {
		if app.MinRetainsBlock < 2 {
			warnRecord(
				"min-retain-blocks should be set to 2 if pruning \"everything\" in app.toml file",
				"set min-retain-blocks to 2",
			)
		}
	} else if app.Pruning == constants.PruningCustom {
		pruningKeepRecent, err := strconv.ParseUint(app.PruningKeepRecent, 10, 64)
		if err != nil {
			exitWithErrorMsgf("ERR: failed to parse pruning-keep-recent in app.toml file: %v\n", err)
			return nil
		}
		if uint64(app.MinRetainsBlock) < pruningKeepRecent {
			warnRecord(
				fmt.Sprintf("min-retain-blocks should be equals to pruning-keep-recent (%s) in app.toml file", app.PruningKeepRecent),
				fmt.Sprintf("set min-retain-blocks to \"%s\"", app.PruningKeepRecent),
			)
		}
	} else if app.Pruning == constants.PruningNothing {
		if app.MinRetainsBlock != 0 {
			fatalRecord(
				"min-retain-blocks must be 0 if pruning \"nothing\" (archival node) in app.toml file",
				"set min-retain-blocks to 0",
			)
		}
	}

	if app.Api == nil {
		exitWithErrorMsgf("ERR: [api] section is missing in app.toml file at %s\n", appTomlFilePath)
		return nil
	}
	if app.Api.Enable {
		if isValidator {
			warnRecord("api is enabled in app.toml file, validator should disable it", "set enable to false")
		}

		if !app.Api.Swagger {
			if isRpc {
				warnRecord("rpc node should enable swagger", "set swagger to true")
			} else if isArchivalNode {
				warnRecord("archival node should enable swagger", "set swagger to true")
			}
		}
	} else {
		if isRpc {
			fatalRecord("api is disabled in app.toml file, rpc node should enable it", "set enable to true")
		} else if isArchivalNode {
			warnRecord("api is disabled in app.toml file, archival node should enable it", "set enable to true")
		}
	}

	if app.JsonRpc != nil {
		if app.JsonRpc.Enable {
			if isValidator {
				warnRecord("json-rpc is enabled in app.toml file, validator should disable it", "set enable to false")
			}
		} else {
			if isRpc {
				fatalRecord("json-rpc is disabled in app.toml file, rpc node should enable it", "set enable to true")
			} else if isArchivalNode {
				warnRecord("json-rpc is disabled in app.toml file, archival node should enable it", "set enable to true")
			}
		}

		if app.JsonRpc.EnableIndexer {
			if isValidator {
				warnRecord(
					"json-rpc custom EVM-indexer is enabled in app.toml file, validator should disable it",
					"set enable-indexer to false",
				)
			}
		} else {
			if isRpc {
				fatalRecord(
					"json-rpc custom EVM-indexer is disabled in app.toml file, rpc node should enable it",
					"set enable-indexer to true",
				)
			} else if isArchivalNode {
				warnRecord(
					"json-rpc custom EVM-indexer is disabled in app.toml file, archival node should enable it",
					"set enable-indexer to true",
				)
			}
		}
	}

	if app.StateSync == nil {
		exitWithErrorMsgf("ERR: [state-sync] section is missing in app.toml file at %s\n", appTomlFilePath)
		return nil
	}
	if app.StateSync.SnapshotInterval == 0 {
		if isRpc {
			warnRecord(
				"snapshot-interval is 0 (disable snapshot) in app.toml file, RPC nodes should set this",
				"set snapshot-interval to 2000",
			)
		} else if isSnapshotNode {
			fatalRecord(
				"snapshot-interval is 0 (disable snapshot) in app.toml file, snapshot nodes must set this",
				"set snapshot-interval to 2000",
			)
		}
	} else {
		if isValidator {
			warnRecord(
				"snapshot-interval is set in app.toml file, validator should not set this",
				"set snapshot-interval to 0 to disable snapshot",
			)
		} else if app.StateSync.SnapshotInterval < 1000 {
			warnRecord(
				"snapshot-interval is too low in app.toml file",
				"set snapshot-interval to 2000",
			)
		}
	}
	if app.StateSync.SnapshotKeepRecent == 0 {
		fatalRecord(
			"snapshot-keep-recent is 0 in app.toml file, means keep all, unset it",
			"set snapshot-keep-recent to 2",
		)
	} else if app.StateSync.SnapshotKeepRecent > 2 {
		warnRecord(
			"snapshot-keep-recent is too high in app.toml file, wasting disk space",
			"set snapshot-keep-recent to 2",
		)
	}

	if app.Grpc == nil {
		exitWithErrorMsgf("ERR: [grpc] section is missing in app.toml file at %s\n", appTomlFilePath)
		return nil
	}
	if app.Grpc.Enable {
		if isValidator {
			warnRecord("grpc is enabled in app.toml file, validator should disable it", "set [grpc] enable to false")
		}
	} else {
		if isValidator {
			// good
		} else if isSnapshotNode {
			// no problem
		} else {
			fatalRecord(
				"grpc is disabled in app.toml file, non-validator node should enable it",
				"set [grpc] enable to true",
			)
		}
	}
	const suggestedMaxSendMsgSizeMb = 100
	const suggestedMaxSendMsgSizeBytes = suggestedMaxSendMsgSizeMb * 1024 * 1024
	maxSendMsgSize, err := strconv.ParseInt(app.Grpc.MaxSendMsgSize, 10, 64)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to parse max-send-msg-size \"%s\" in app.toml file: %v\n", app.Grpc.MaxSendMsgSize, err)
		return nil
	}
	if maxSendMsgSize < suggestedMaxSendMsgSizeBytes {
		warnRecord(
			"max-send-msg-size is too low in app.toml file",
			fmt.Sprintf("set max-send-msg-size to %d (%d MB)", suggestedMaxSendMsgSizeBytes, suggestedMaxSendMsgSizeMb),
		)
	}
	if isRpc || isArchivalNode {
		if app.Grpc.Enable {
			if maxSendMsgSize > suggestedMaxSendMsgSizeBytes*5 {
				warnRecord(
					"max-send-msg-size is too high in app.toml file",
					fmt.Sprintf("set max-send-msg-size to %d (%d MB)", suggestedMaxSendMsgSizeBytes, suggestedMaxSendMsgSizeMb),
				)
			}
			if strings.HasSuffix(app.Grpc.Address, ":9090") {
				warnRecord(
					"GRPC port should not be the default one (9090) on RPC and Archival node",
					"set [grpc] address to a custom port",
				)
			}
		}
	}

	return &app
}

func checkHomeConfigClientToml(configPath string) {
	clientTomlFilePath := path.Join(configPath, "client.toml")
	perm, exists, isDir, err := utils.FileInfo(clientTomlFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check client.toml file at %s: %v\n", clientTomlFilePath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: client.toml file does not exist: %s\n", clientTomlFilePath)
		return
	}
	if isDir {
		exitWithErrorMsgf("ERR: client.toml is a directory, it should be a file: %s\n", clientTomlFilePath)
		return
	}
	filePerm := types.FilePermFrom(perm)
	if filePerm.Other.AnyPermission() {
		fatalRecord("client.toml file is accessible by others", "chmod 600 "+clientTomlFilePath)
	}
	if filePerm.Group.AnyPermission() {
		fatalRecord("client.toml file is accessible by group", "chmod 600 "+clientTomlFilePath)
	}
	if !filePerm.User.Read {
		fatalRecord("client.toml file is not readable by user", "chmod 600 "+clientTomlFilePath)
	}
	if !filePerm.User.Write {
		fatalRecord("client.toml file is not writable by user", "chmod 600 "+clientTomlFilePath)
	}
}

func checkHomeConfigConfigToml(configPath string, nodeType types.NodeType) *types.ConfigToml {
	isValidator := nodeType == types.ValidatorNode
	configTomlFilePath := path.Join(configPath, "config.toml")
	perm, exists, isDir, err := utils.FileInfo(configTomlFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check config.toml file at %s: %v\n", configTomlFilePath, err)
		return nil
	}
	if !exists {
		exitWithErrorMsgf("ERR: config.toml file does not exist: %s\n", configTomlFilePath)
		return nil
	}
	if isDir {
		exitWithErrorMsgf("ERR: config.toml is a directory, it should be a file: %s\n", configTomlFilePath)
		return nil
	}
	filePerm := types.FilePermFrom(perm)
	if filePerm.Other.Write {
		fatalRecord("config.toml file is writable by others", "chmod 644 "+configTomlFilePath)
	}
	if filePerm.Group.Write {
		fatalRecord("config.toml file is writable by group", "chmod 644 "+configTomlFilePath)
	}
	if !filePerm.User.Read {
		fatalRecord("config.toml file is not readable by user", "chmod 644 "+configTomlFilePath)
	}
	if !filePerm.User.Write {
		fatalRecord("config.toml file is not writable by user", "chmod 644 "+configTomlFilePath)
	}

	bz, err := os.ReadFile(configTomlFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to read config.toml file at %s: %v\n", configTomlFilePath, err)
		return nil
	}

	var config types.ConfigToml
	err = toml.Unmarshal(bz, &config)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to unmarshal config.toml file at %s: %v\n", configTomlFilePath, err)
		return nil
	}

	if config.Moniker == "" {
		fatalRecord("moniker is empty in config.toml file", "set moniker to a unique name")
	}

	if config.P2P == nil {
		exitWithErrorMsgf("ERR: [p2p] section is missing in config.toml file at %s\n", configTomlFilePath)
		return nil
	}
	if config.P2P.Seeds == "" {
		warnRecord("seeds is empty in config.toml file", "set seeds to seed nodes")
	} else if !isValidPeer(config.P2P.Seeds) {
		warnRecord("invalid seeds format in config.toml file", "correct the format of seeds")
	}
	if strings.HasSuffix(config.P2P.Laddr, ":26656") {
		if isValidator {
			warnRecord("P2P port should not be the default one (26656) on validator node", "set p2p laddr to a custom port")
		} else {
			warnRecord("P2P port should not be the default one (26656)", "set p2p laddr to a custom port")
		}
	}
	if config.P2P.PersistentPeers == "" {
		warnRecord("persistent_peers is empty in config.toml file", "set persistent_peers to persistent peer nodes")
	} else if !isValidPeer(config.P2P.PersistentPeers) {
		warnRecord("invalid persistent_peers format in config.toml file", "correct the format of persistent_peers")
	}
	if config.P2P.MaxNumInboundPeers < 60 {
		warnRecord("max_num_inbound_peers is too low in config.toml file", "increase max_num_inbound_peers to 120")
	}
	if config.P2P.MaxNumOutboundPeers <= 30 {
		warnRecord("max_num_outbound_peers is too low in config.toml file", "increase max_num_outbound_peers to 60")
	}
	if config.P2P.SeedMode {
		warnRecord("seed_mode is enabled in config.toml file", "disable seed_mode if not on purpose")
	}

	if config.StateSync == nil {
		exitWithErrorMsgf("ERR: [statesync] section is missing in config.toml file at %s\n", configTomlFilePath)
		return nil
	}
	if config.StateSync.Enable {
		warnRecord("statesync is enabled in config.toml file", "disable state sync in section [statesync]")
	}

	if config.Consensus == nil {
		exitWithErrorMsgf("ERR: [consensus] section is missing in config.toml file at %s\n", configTomlFilePath)
		return nil
	}
	if config.Consensus.DoubleSignCheckHeight > 0 {
		if isValidator {
			if config.Consensus.DoubleSignCheckHeight > constants.MaxDoubleSignCheckHeight {
				warnRecord(
					fmt.Sprintf("double_sign_check_height %d is too high in config.toml file, can lower uptime", config.Consensus.DoubleSignCheckHeight),
					fmt.Sprintf("set double_sign_check_height to %d", constants.RecommendDoubleSignCheckHeight),
				)
			} else if config.Consensus.DoubleSignCheckHeight < constants.MinDoubleSignCheckHeight {
				warnRecord(
					fmt.Sprintf("double_sign_check_height %d is too low in config.toml file", config.Consensus.DoubleSignCheckHeight),
					fmt.Sprintf("set double_sign_check_height to %d", constants.RecommendDoubleSignCheckHeight),
				)
			}
		}
	} else {
		if isValidator {
			fatalRecord(
				"double_sign_check_height is not set in config.toml file, validator nodes should set this",
				fmt.Sprintf("set double_sign_check_height to %d", constants.RecommendDoubleSignCheckHeight),
			)
		}
	}
	if config.Consensus.SkipTimeoutCommit {
		if isValidator {
			fatalRecord(
				"skip_timeout_commit is enabled in config.toml file, validator nodes should not use this",
				"disable skip_timeout_commit",
			)
		} else {
			warnRecord(
				"skip_timeout_commit is enabled in config.toml file",
				"disable skip_timeout_commit",
			)
		}
	}

	if config.TxIndex == nil {
		exitWithErrorMsgf("ERR: [tx_index] section is missing in config.toml file at %s\n", configTomlFilePath)
		return nil
	}
	switch config.TxIndex.Indexer {
	case "":
		if isValidator {
			fatalRecord(
				"indexer is empty in [tx_index] section of config.toml file, validator nodes should set this to \"null\"",
				"set indexer to \"null\"",
			)
		} else {
			warnRecord(
				"indexer is empty in [tx_index] section of config.toml file, non-validator nodes should set this to \"kv\"",
				"set indexer to \"kv\"",
			)
		}
	case "kv":
		if isValidator {
			warnRecord(
				"indexer is set to \"kv\" in [tx_index] section of config.toml file, validator nodes should set this to \"null\"",
				"set indexer to \"null\"",
			)
		}
	case "null":
		if !isValidator {
			fatalRecord(
				"indexer is set to \"null\" (disable indexer) in [tx_index] section of config.toml file, non-validator nodes should set this to \"kv\"",
				"set indexer to \"kv\"",
			)
		}
	default:
		if isValidator {
			fatalRecord(
				fmt.Sprintf("invalid indexer option \"%s\" in [tx_index] section of config.toml file", config.TxIndex.Indexer),
				"set indexer to \"null\"",
			)
		} else {
			fatalRecord(
				fmt.Sprintf("invalid indexer option \"%s\" in [tx_index] section of config.toml file", config.TxIndex.Indexer),
				"set indexer to \"kv\"",
			)
		}
	}

	return &config
}

func checkHomeConfigGenesisJson(configPath string) {
	genesisJsonFilePath := path.Join(configPath, "genesis.json")
	perm, exists, isDir, err := utils.FileInfo(genesisJsonFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check genesis.json file at %s: %v\n", genesisJsonFilePath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: genesis.json file does not exist: %s\n", genesisJsonFilePath)
		return
	}
	if isDir {
		exitWithErrorMsgf("ERR: genesis.json is a directory, it should be a file: %s\n", genesisJsonFilePath)
		return
	}
	filePerm := types.FilePermFrom(perm)
	if filePerm.Other.Write {
		fatalRecord("genesis.json file is writable by others", "chmod 644 "+genesisJsonFilePath)
	}
	if filePerm.Group.Write {
		fatalRecord("genesis.json file is writable by group", "chmod 644 "+genesisJsonFilePath)
	}
	if !filePerm.User.Read {
		fatalRecord("genesis.json file is not readable by user", "chmod 644 "+genesisJsonFilePath)
	}
	if !filePerm.User.Write {
		fatalRecord("genesis.json file is not writable by user", "chmod 644 "+genesisJsonFilePath)
	}
}

func checkHomeConfigNodeKeyJson(configPath string) {
	nodeKeyJsonFilePath := path.Join(configPath, "node_key.json")
	perm, exists, isDir, err := utils.FileInfo(nodeKeyJsonFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check node_key.json file at %s: %v\n", nodeKeyJsonFilePath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: node_key.json file does not exist: %s\n", nodeKeyJsonFilePath)
		return
	}
	if isDir {
		exitWithErrorMsgf("ERR: node_key.json is a directory, it should be a file: %s\n", nodeKeyJsonFilePath)
		return
	}
	filePerm := types.FilePermFrom(perm)
	if filePerm.Other.AnyPermission() {
		fatalRecord("node_key.json file is accessible by others", "chmod 600 "+nodeKeyJsonFilePath)
	}
	if filePerm.Group.AnyPermission() {
		fatalRecord("node_key.json file is accessible by group", "chmod 600 "+nodeKeyJsonFilePath)
	}
	if !filePerm.User.Read {
		fatalRecord("node_key.json file is not readable by user", "chmod 600 "+nodeKeyJsonFilePath)
	}
	if !filePerm.User.Write {
		fatalRecord("node_key.json file is not writable by user", "chmod 600 "+nodeKeyJsonFilePath)
	}

	type nodeKeyPrivKey struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type nodeKey struct {
		PrivKey *nodeKeyPrivKey `json:"priv_key"`
	}

	bz, err := os.ReadFile(nodeKeyJsonFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to read node_key.json file at %s: %v\n", nodeKeyJsonFilePath, err)
		return
	}

	if len(bz) == 0 {
		exitWithErrorMsgf("ERR: node_key.json file is empty: %s\n", nodeKeyJsonFilePath)
		return
	}

	var nk nodeKey
	err = json.Unmarshal(bz, &nk)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to unmarshal node_key.json file at %s: %v\n", nodeKeyJsonFilePath, err)
		return
	}

	if nk.PrivKey == nil {
		exitWithErrorMsgf("ERR: priv_key is missing in node_key.json file at %s\n", nodeKeyJsonFilePath)
		return
	}

	if len(nk.PrivKey.Type) == 0 {
		exitWithErrorMsgf("ERR: type is missing in priv_key in node_key.json file at %s\n", nodeKeyJsonFilePath)
		return
	}

	if len(nk.PrivKey.Value) == 0 {
		exitWithErrorMsgf("ERR: value is missing in priv_key in node_key.json file at %s\n", nodeKeyJsonFilePath)
		return
	}
}

func checkHomeConfigPrivValidatorKeyJson(configPath string) {
	privValidatorJsonFilePath := path.Join(configPath, "priv_validator_key.json")
	perm, exists, isDir, err := utils.FileInfo(privValidatorJsonFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check priv_validator_key.json file at %s: %v\n", privValidatorJsonFilePath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: priv_validator_key.json file does not exist: %s\n", privValidatorJsonFilePath)
		return
	}
	if isDir {
		exitWithErrorMsgf("ERR: priv_validator_key.json is a directory, it should be a file: %s\n", privValidatorJsonFilePath)
		return
	}
	filePerm := types.FilePermFrom(perm)
	if filePerm.Other.AnyPermission() {
		fatalRecord("priv_validator_key.json file is accessible by others", "chmod 600 "+privValidatorJsonFilePath)
	}
	if filePerm.Group.AnyPermission() {
		fatalRecord("priv_validator_key.json file is accessible by group", "chmod 600 "+privValidatorJsonFilePath)
	}
	if !filePerm.User.Read {
		fatalRecord("priv_validator_key.json file is not readable by user", "chmod 600 "+privValidatorJsonFilePath)
	}
	if !filePerm.User.Write {
		fatalRecord("priv_validator_key.json file is not writable by user", "chmod 600 "+privValidatorJsonFilePath)
	}

	type privKey struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type pubKey struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type privValidatorKey struct {
		PrivKey *privKey `json:"priv_key"`
		PubKey  *pubKey  `json:"pub_key"`
		Address string   `json:"address"`
	}

	bz, err := os.ReadFile(privValidatorJsonFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to read priv_validator_key.json file at %s: %v\n", privValidatorJsonFilePath, err)
		return
	}

	if len(bz) == 0 {
		exitWithErrorMsgf("ERR: priv_validator_key.json file is empty: %s\n", privValidatorJsonFilePath)
		return
	}

	var nk privValidatorKey
	err = json.Unmarshal(bz, &nk)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to unmarshal priv_validator_key.json file at %s: %v\n", privValidatorJsonFilePath, err)
		return
	}

	if nk.PrivKey == nil {
		exitWithErrorMsgf("ERR: priv_key is missing in priv_validator_key.json file at %s\n", privValidatorJsonFilePath)
		return
	}

	if len(nk.PrivKey.Type) == 0 {
		exitWithErrorMsgf("ERR: type is missing in priv_key in priv_validator_key.json file at %s\n", privValidatorJsonFilePath)
		return
	}

	if len(nk.PrivKey.Value) == 0 {
		exitWithErrorMsgf("ERR: value is missing in priv_key in priv_validator_key.json file at %s\n", privValidatorJsonFilePath)
		return
	}

	if nk.PubKey == nil {
		exitWithErrorMsgf("ERR: pub_key is missing in priv_validator_key.json file at %s\n", privValidatorJsonFilePath)
		return
	}

	if len(nk.PubKey.Type) == 0 {
		exitWithErrorMsgf("ERR: type is missing in pub_key in priv_validator_key.json file at %s\n", privValidatorJsonFilePath)
		return
	}

	if len(nk.PubKey.Value) == 0 {
		exitWithErrorMsgf("ERR: value is missing in pub_key in priv_validator_key.json file at %s\n", privValidatorJsonFilePath)
		return
	}

	if len(nk.Address) == 0 {
		exitWithErrorMsgf("ERR: address is missing in priv_validator_key.json file at %s\n", privValidatorJsonFilePath)
		return
	}

	if !regexp.MustCompile(`^[\dA-F]{40}$`).MatchString(nk.Address) {
		exitWithErrorMsgf("ERR: address is malformed in priv_validator_key.json file at %s\n", privValidatorJsonFilePath)
		return
	}
}

func checkHomeConfigConfigTomlAndAppToml(nodeType types.NodeType, configToml *types.ConfigToml, appToml *types.AppToml) {
	if configToml == nil || appToml == nil {
		panic("configToml or appToml is nil")
	}

	isValidator := nodeType == types.ValidatorNode

	if isValidator {
		if appToml.Pruning == constants.PruningCustom {
			if appToml.PruningKeepRecent != "" {
				pruningKeepRecent, err := strconv.ParseUint(appToml.PruningKeepRecent, 10, 64)
				if err != nil {
					exitWithErrorMsgf("ERR: failed to parse pruning-keep-recent in app.toml file: %v\n", err)
					return
				}

				if pruningKeepRecent <= uint64(configToml.Consensus.DoubleSignCheckHeight) {
					warnRecord(
						fmt.Sprintf(
							"pruning-keep-recent %d should be greater than double_sign_check_height %d in app.toml file",
							pruningKeepRecent,
							configToml.Consensus.DoubleSignCheckHeight,
						),
						fmt.Sprintf("increase pruning-keep-recent to be greater than double_sign_check_height few blocks"),
					)
				}
			}

			if appToml.MinRetainsBlock <= configToml.Consensus.DoubleSignCheckHeight {
				warnRecord(
					fmt.Sprintf(
						"min-retain-blocks %d should be greater than double_sign_check_height %d in app.toml file",
						appToml.MinRetainsBlock,
						configToml.Consensus.DoubleSignCheckHeight,
					),
					fmt.Sprintf("increase min-retain-blocks to be greater than double_sign_check_height few blocks"),
				)
			}
		}
	}
}
