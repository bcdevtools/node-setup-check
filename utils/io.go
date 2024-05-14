package utils

import "os"

func FileInfo(path string) (fileMode os.FileMode, exists, isDir bool, error error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		error = err
		return
	}

	fileMode = fi.Mode().Perm()
	exists = true
	isDir = fi.IsDir()
	return
}
