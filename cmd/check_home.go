package cmd

import (
	"fmt"
	"github.com/bcdevtools/node-setup-check/types"
	"github.com/bcdevtools/node-setup-check/utils"
)

func checkHome(home string) {
	perm, exists, isDir, err := utils.FileInfo(home)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check provided home directory: %s\n", err)
		return
	}
	if !exists {
		exitWithErrorMsg("ERR: provided home directory does not exist")
		return
	}
	if !isDir {
		exitWithErrorMsg("ERR: provided home directory is not a directory")
		return
	}

	filePerm := types.FilePermFrom(perm)
	if filePerm.Other.Write {
		fatalRecord("home directory is writable by others", fmt.Sprintf("chmod o-w %s", home))
	}
	if filePerm.Group.Write {
		fatalRecord("home directory is writable by group", fmt.Sprintf("chmod g-w %s", home))
	}
	if !filePerm.User.IsFullPermission() {
		fatalRecord("home directory is fully accessible by user", fmt.Sprintf("chmod u+rwx %s", home))
	}
}
