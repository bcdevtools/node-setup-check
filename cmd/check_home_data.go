package cmd

import (
	"encoding/json"
	"github.com/EscanBE/node-setup-check/types"
	"github.com/EscanBE/node-setup-check/utils"
	"os"
	"path"
)

func checkHomeData(home string, nodeType types.NodeType) {
	dataPath := path.Join(home, "data")
	perm, exists, isDir, err := utils.FileInfo(dataPath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check data directory at %s: %v\n", dataPath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: data directory does not exist: %s\n", dataPath)
		return
	}
	if !isDir {
		exitWithErrorMsgf("ERR: data is not a directory: %s\n", dataPath)
		return
	}

	filePerm := types.FilePermFrom(perm)
	if filePerm.Other.AnyPermission() {
		fatalRecord("data directory is accessible by others", "chmod 700 "+dataPath)
	}
	if filePerm.Group.AnyPermission() {
		fatalRecord("data directory is accessible by group", "chmod 700 "+dataPath)
	}
	if !filePerm.User.IsFullPermission() {
		fatalRecord("data directory is not fully accessible by user", "chmod 700 "+dataPath)
	}

	privValidatorStateFilePath := path.Join(dataPath, "priv_validator_state.json")
	perm, exists, isDir, err = utils.FileInfo(privValidatorStateFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check priv_validator_state.json file at %s: %v\n", privValidatorStateFilePath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: priv_validator_state.json file is missing")
		return
	}
	if isDir {
		exitWithErrorMsgf("ERR: priv_validator_state.json is a directory, it should be a file")
		return
	}
	if perm != 0o600 {
		fatalRecord("priv_validator_state.json has invalid permission", "chmod 600 "+privValidatorStateFilePath)
	}

	type privateValidatorState struct {
		Height    string `json:"height"`
		Round     int    `json:"round"`
		Step      int    `json:"step"`
		Signature string `json:"signature"`
		SignBytes string `json:"signbytes"`
	}

	var pvs privateValidatorState
	bz, err := os.ReadFile(privValidatorStateFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to read priv_validator_state.json file: %v\n", err)
		return
	}

	err = json.Unmarshal(bz, &pvs)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to unmarshal priv_validator_state.json file: %v\n", err)
		return
	}

	if pvs.Height == "0" && pvs.Round == 0 && pvs.Step == 0 && pvs.Signature == "" && pvs.SignBytes == "" {
		// empty
		if nodeType == types.ValidatorNode {
			fatalRecord("priv_validator_state.json is empty", "can be ignored if this is a fresh validator node")
		}
	} else {
		if nodeType == types.ValidatorNode {
			if pvs.Height == "0" {
				exitWithErrorMsg("priv_validator_state.json is not empty, but height is 0, trouble-shoot the issue")
				return
			}
			if pvs.Signature == "" {
				exitWithErrorMsg("priv_validator_state.json is not empty, but signature is empty, trouble-shoot the issue")
				return
			}
			if pvs.SignBytes == "" {
				exitWithErrorMsg("priv_validator_state.json is not empty, but signbytes is empty, trouble-shoot the issue")
				return
			}
		} else {
			exitWithErrorMsg("priv_validator_state.json is not empty, it should be empty on non-validator nodes, trouble-shoot the issue")
			return
		}
	}
}
