package cmd

import (
	"fmt"
	"github.com/bcdevtools/node-setup-check/types"
	"github.com/bcdevtools/node-setup-check/utils"
	"github.com/pkg/errors"
	"os"
	"path"
	"path/filepath"
)

func checkHomeKeyring(home string, isValidatorNode bool) {
	checkHomeKeyringFile(home, isValidatorNode)
	checkHomeKeyringTest(home, isValidatorNode)
}

func checkHomeKeyringFile(home string, isValidatorNode bool) {
	keyringFilePath := path.Join(home, "keyring-file")
	perm, exists, isDir, err := utils.FileInfo(keyringFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check keyring-file directory at %s: %v\n", keyringFilePath, err)
		return
	}

	if !exists {
		if isValidatorNode {
			warnRecord(fmt.Sprintf("keyring-file directory is missing on validator node: %s", keyringFilePath), "can be ignored if you are not using keyring-file")
		}
		return
	}

	if !isDir {
		exitWithErrorMsgf("ERR: keyring-file is not a directory: %s", keyringFilePath)
		return
	}

	if !isValidatorNode {
		isEmpty, err := isEmptyDir(keyringFilePath)
		if err != nil {
			exitWithErrorMsgf("ERR: failed to check emptiness of keyring-file directory at %s: %v\n", keyringFilePath, err)
			return
		}
		if !isEmpty {
			warnRecord(fmt.Sprintf("should not store key on non-validator node, found at %s", keyringFilePath), "migrate/backup and remove usage of keyring-file")
		}
	}

	if perm != 0o700 {
		fatalRecord(fmt.Sprintf("keyring-file directory has invalid permission %s", perm.String()), fmt.Sprintf("chmod -R 700 %s", keyringFilePath))
	}

	// check file hash
	fileHashPath := path.Join(keyringFilePath, "keyhash")
	perm, exists, isDir, err = utils.FileInfo(fileHashPath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check %s: %v\n", fileHashPath, err)
		return
	}
	if exists {
		if isDir {
			exitWithErrorMsgf("ERR: %s is a directory, it should be a file\n", fileHashPath)
			return
		}

		filePerm := types.FilePermFrom(perm)
		if filePerm.Other.AnyPermission() {
			fatalRecord("keyhash file should not be accessible by others", fmt.Sprintf("chmod 600 %s", fileHashPath))
		}
		if filePerm.Group.AnyPermission() {
			fatalRecord("keyhash file should not be accessible by group", fmt.Sprintf("chmod 600 %s", fileHashPath))
		}
		if !filePerm.User.Read {
			fatalRecord("keyhash file should be readable by owner", fmt.Sprintf("chmod 600 %s", fileHashPath))
		}
		if !filePerm.User.Write {
			fatalRecord("keyhash file should be writable by owner", fmt.Sprintf("chmod 600 %s", fileHashPath))
		}
	} else if isValidatorNode {
		warnRecord(fmt.Sprintf("keyhash file is missing on validator node: %s", fileHashPath), "can be ignored if you are not using keyring-file")
	}

	err = filepath.Walk(keyringFilePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == keyringFilePath {
			return nil
		}

		perm, _, isDir, err = utils.FileInfo(path)
		if err != nil {
			return errors.Wrapf(err, "failed to check keyring-file inner file %s", path)
		}

		if isDir && perm != 0o700 {
			fatalRecord(fmt.Sprintf("keyring-file inner directory must have permission 700 but %s has invalid permission %s", path, perm.String()), fmt.Sprintf("chmod -R 700 %s", keyringFilePath))
		} else if !isDir && perm != 0o600 && perm != 0o700 {
			fatalRecord(fmt.Sprintf("keyring-file inner file must have permission 600 or 700 but %s has invalid permission %s", path, perm.String()), fmt.Sprintf("chmod -R 600 %s", keyringFilePath))
		}

		return nil
	})

	if err != nil {
		exitWithErrorMsgf("ERR: failed to walk on %s: %v\n", keyringFilePath, err)
		return
	}
}

func checkHomeKeyringTest(home string, isValidatorNode bool) {
	keyringTestPath := path.Join(home, "keyring-test")
	perm, exists, isDir, err := utils.FileInfo(keyringTestPath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check keyring-test directory at %s: %v\n", keyringTestPath, err)
		return
	}

	if !exists {
		return
	}

	if !isDir {
		return
	}

	if perm != 0o700 {
		fatalRecord(fmt.Sprintf("keyring-test directory has invalid permission %s", perm.String()), fmt.Sprintf("chmod -R 700 %s", keyringTestPath))
	}

	isEmpty, err := isEmptyDir(keyringTestPath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check emptiness of keyring-test directory at %s: %v\n", keyringTestPath, err)
		return
	}

	if !isEmpty {
		if isValidatorNode {
			exitWithErrorMsgf("ERR: keyring-test directory is found on validator node: %s ! Migrate/backup and remove usage of keyring-test\n> rm -rf %s", keyringTestPath, keyringTestPath)
			return
		}
		fatalRecord("keyring-test should not be used, found at "+keyringTestPath, "migrate/backup and remove usage of keyring-test")
	}

	err = filepath.Walk(keyringTestPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == keyringTestPath {
			return nil
		}

		perm, _, isDir, err = utils.FileInfo(path)
		if err != nil {
			return errors.Wrapf(err, "failed to check keyring-test inner file %s", path)
		}

		if isDir && perm != 0o700 {
			fatalRecord(fmt.Sprintf("keyring-test inner directory must have permission 700 but %s has invalid permission %s", path, perm.String()), fmt.Sprintf("chmod -R 700 %s", keyringTestPath))
		} else if !isDir && perm != 0o600 && perm != 0o700 {
			fatalRecord(fmt.Sprintf("keyring-test inner file must have permission 600 or 700 but %s has invalid permission %s", path, perm.String()), fmt.Sprintf("chmod -R 600 %s", keyringTestPath))
		}

		return nil
	})

	if err != nil {
		exitWithErrorMsgf("ERR: failed to walk on %s: %v\n", keyringTestPath, err)
		return
	}
}
