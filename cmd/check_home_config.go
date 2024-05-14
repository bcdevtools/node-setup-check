package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/EscanBE/node-setup-check/types"
	"github.com/EscanBE/node-setup-check/utils"
	"github.com/pelletier/go-toml/v2"
	"os"
	"path"
	"regexp"
	"strconv"
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

	checkHomeConfigAppToml(configPath, nodeType)
	checkHomeConfigClientToml(configPath, nodeType)
	checkHomeConfigConfigToml(configPath, nodeType)
	checkHomeConfigGenesisJson(configPath, nodeType)
	checkHomeConfigNodeKeyJson(configPath, nodeType)
	checkHomeConfigPrivValidatorKeyJson(configPath, nodeType)
}

func checkHomeConfigAppToml(configPath string, nodeType types.NodeType) {
	isValidator := nodeType == types.ValidatorNode
	isRpc := nodeType == types.RpcNode
	isSnapshotNode := nodeType == types.SnapshotNode
	isArchivalNode := nodeType == types.ArchivalNode
	appTomlFilePath := path.Join(configPath, "app.toml")
	perm, exists, isDir, err := utils.FileInfo(appTomlFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check app.toml file at %s: %v\n", appTomlFilePath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: app.toml file does not exist: %s\n", appTomlFilePath)
		return
	}
	if isDir {
		exitWithErrorMsgf("ERR: app.toml is a directory, it should be a file: %s\n", appTomlFilePath)
		return
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
		return
	}

	type apiAppToml struct {
		Enable bool `toml:"enable"`
	}
	type jsonRpcAppToml struct {
		Enable bool `toml:"enable"`
	}
	type appToml struct {
		MinimumGasPrices  string          `toml:"minimum-gas-prices"`
		Pruning           string          `toml:"pruning"`
		PruningKeepRecent string          `toml:"pruning-keep-recent"`
		PruningInterval   string          `toml:"pruning-interval"`
		HaltHeight        int64           `toml:"halt-height"`
		Api               *apiAppToml     `toml:"api"`
		JsonRpc           *jsonRpcAppToml `toml:"json-rpc"`
	}

	var app appToml
	err = toml.Unmarshal(bz, &app)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to unmarshal app.toml file at %s: %v\n", appTomlFilePath, err)
		return
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
	case "default":
		if isValidator {
			warnRecord("pruning set to 'default' in app.toml file", "set pruning to everything for validator")
		} else if isSnapshotNode {
			warnRecord("pruning set to 'default' in app.toml file, snapshot not should be configured properly for snapshot purpose", "set pruning to custom 100/10")
		} else if isArchivalNode {
			fatalRecord("pruning set to 'default' in app.toml file, archival node should be configured properly for archival purpose", "set pruning to nothing")
		}
	case "nothing":
		if isValidator {
			fatalRecord("pruning set to 'nothing' in app.toml file, validator should not use this option", "set pruning to everything")
		} else if isSnapshotNode {
			fatalRecord("pruning set to 'nothing' in app.toml file, snapshot not should be configured properly for snapshot purpose", "set pruning to custom 100/10")
		}
	case "everything":
		if isValidator {
			//
		} else if isArchivalNode {
			fatalRecord("pruning set to 'everything' in app.toml file, archival node must not use this option", "set pruning to nothing")
		} else {
			fatalRecord("pruning set to 'everything' in app.toml file, non-validator should not use this option", "set pruning to default or custom")
		}
	case "custom":
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
	}

	if isSnapshotNode {
		if app.Pruning != "custom" || app.PruningKeepRecent != "100" || app.PruningInterval != "10" {
			warnRecord("snapshot node should use pruning custom 100/10 in app.toml file", "set pruning to custom 100/10")
		}
	}

	if app.Pruning == "custom" {
		if app.PruningKeepRecent != "" {
			pruningKeepRecent, err := strconv.ParseInt(app.PruningKeepRecent, 10, 64)
			if err != nil {
				exitWithErrorMsgf("ERR: failed to parse pruning-keep-recent in app.toml file: %v\n", err)
				return
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
				return
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
		warnRecord(fmt.Sprintf("halt-height is set to %d in app.toml file", app.HaltHeight), "unset halt-height")
	}

	if app.Api == nil {
		exitWithErrorMsgf("ERR: [api] section is missing in app.toml file at %s\n", appTomlFilePath)
		return
	}
	if app.Api.Enable {
		if isValidator {
			warnRecord("api is enabled in app.toml file, validator should disable it", "set enable to false")
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
	}
}

func checkHomeConfigClientToml(configPath string, nodeType types.NodeType) {
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

func checkHomeConfigConfigToml(configPath string, nodeType types.NodeType) {
	configTomlFilePath := path.Join(configPath, "config.toml")
	perm, exists, isDir, err := utils.FileInfo(configTomlFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check config.toml file at %s: %v\n", configTomlFilePath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: config.toml file does not exist: %s\n", configTomlFilePath)
		return
	}
	if isDir {
		exitWithErrorMsgf("ERR: config.toml is a directory, it should be a file: %s\n", configTomlFilePath)
		return
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
}

func checkHomeConfigGenesisJson(configPath string, nodeType types.NodeType) {
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

func checkHomeConfigNodeKeyJson(configPath string, nodeType types.NodeType) {
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

func checkHomeConfigPrivValidatorKeyJson(configPath string, nodeType types.NodeType) {
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
