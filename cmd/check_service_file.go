package cmd

import (
	"fmt"
	"github.com/bcdevtools/node-setup-check/utils"
	"github.com/sergeymakinen/go-systemdconf/v2"
	"github.com/sergeymakinen/go-systemdconf/v2/unit"
	"os"
	"path/filepath"
	"strings"
)

func checkServiceFileForValidatorOnLinux(home string, serviceFilePath string) {
	perm, exists, isDir, err := utils.FileInfo(serviceFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check service file at %s: %v\n", serviceFilePath, err)
		return
	}
	if !exists {
		exitWithErrorMsgf("ERR: service file does not exist: %s\n", serviceFilePath)
		return
	}
	if isDir {
		exitWithErrorMsgf("ERR: service file is a directory, it should be a file: %s\n", serviceFilePath)
		return
	}
	if perm != 0o644 {
		fatalRecord("service file has invalid permission", "sudo chmod 644 "+serviceFilePath)
	}
	if !strings.HasSuffix(serviceFilePath, ".service") {
		fatalRecord("service file is not a systemd service file", "use .service file extension")
	}
	if !strings.HasPrefix(serviceFilePath, "/etc/systemd/system") {
		warnRecord("service file is not in /etc/systemd/system directory", "use systemd")
	}

	// check service file content

	bz, err := os.ReadFile(serviceFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to read service file: %v\n", err)
		return
	}

	var sf unit.ServiceFile
	err = systemdconf.Unmarshal(bz, &sf)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to unmarshal service file: %v\n", err)
		return
	}

	originalRecordsCount := len(checkRecords)
	defer func() {
		if len(checkRecords) > originalRecordsCount {
			warnRecord("remember to reload service after updated service file", "sudo systemctl daemon-reload")
		}
	}()

	if sf.Unit.Description.String() == "" {
		fatalRecord("service file is missing Description in [Unit] section", "add Description to [Unit] section")
	}
	if sf.Unit.After.String() == "" {
		fatalRecord("service file is missing After in [Unit] section", "add After to [Unit] section")
	} else if sf.Unit.After.String() != "network-online.target" {
		fatalRecord("service file is using invalid After in [Unit] section", "change After to network-online.target")
	}

	if sf.Service.User.String() == "" {
		fatalRecord("service file is missing User in [Service] section", "add User to [Service] section")
	} else {
		user := strings.TrimSpace(strings.ToLower(sf.Service.User.String()))
		if user == "root" || user == "ubuntu" {
			fatalRecord(
				"service file is using invalid User in [Service] section",
				"change User to a non-root user",
			)
		} else if !strings.Contains(user, "-") {
			warnRecord(
				"service file is using invalid User in [Service] section",
				"use memorable username with hyphen, e.g. \"val-x-testnet\"",
			)
		}
	}
	if sf.Service.ExecStart.String() == "" {
		fatalRecord(
			"service file is missing ExecStart in [Service] section", "add ExecStart to [Service] section",
		)
	} else if !strings.Contains(sf.Service.ExecStart.String(), "--home") {
		fatalRecord(
			"service file is missing --home in ExecStart in [Service] section",
			"add --home to ExecStart in [Service] section",
		)
	} else {
		_, homeName := filepath.Split(home)
		if !strings.Contains(sf.Service.ExecStart.String(), homeName) {
			fatalRecord(
				fmt.Sprintf("--home in ExecStart in [Service] section might not pointing to the correct home dir \"%s\"", homeName),
				"change --home to --home="+homeName,
			)
		}
	}
	if sf.Service.Restart.String() == "" {
		fatalRecord(
			"service file is missing Restart in [Service] section",
			"add Restart=no to [Service] section",
		)
	} else if sf.Service.Restart.String() != "no" {
		fatalRecord(
			"service file is using invalid Restart in [Service] section, must using 'no' to prevent incident restart",
			"change Restart=no",
		)
	}
	if sf.Service.RestartSec.String() != "" {
		fatalRecord(
			"service file contains RestartSec in [Service] section",
			"remove RestartSec from [Service] section",
		)
	}

	if sf.Install.WantedBy.String() == "" {
		fatalRecord(
			"service file is missing WantedBy in [Install] section",
			"add WantedBy=multi-user.target in [Install] section",
		)
	} else if sf.Install.WantedBy.String() != "multi-user.target" {
		fatalRecord(
			"service file is using invalid WantedBy in [Install] section",
			"change WantedBy to multi-user.target in [Install] section",
		)
	}

	_, serviceFileName := filepath.Split(serviceFilePath)
	multiUserTargetWantsServiceFilePath := filepath.Join("/etc/systemd/system/multi-user.target.wants", serviceFileName)
	_, exists, _, err = utils.FileInfo(multiUserTargetWantsServiceFilePath)
	if err != nil {
		exitWithErrorMsgf("ERR: failed to check if service file is enabled: %v\n", err)
		return
	}
	if exists {
		fatalRecord(
			"service file is already enabled, validator must disable service automatically run at startup",
			"sudo systemctl disable "+serviceFileName,
		)
	}
}
