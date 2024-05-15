package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/bcdevtools/node-setup-check/constants"
	"github.com/bcdevtools/node-setup-check/types"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	flagType        = "type"
	flagServiceFile = "service-file"
)

var waitGroup sync.WaitGroup

func GetCheckCmd() *cobra.Command {
	validTargetValues := strings.Join(types.AllNodeTypeNames(), "/")

	var cmd = &cobra.Command{
		Use:     "check [home]",
		Aliases: []string{},
		Args:    cobra.ExactArgs(1),
		Short:   "Check node setup",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("App version", constants.VERSION)
			fmt.Println("NOTICE: always update to latest version for accurate check")
			go checkLatestRelease()
			time.Sleep(2 * time.Second)

			typeName, _ := cmd.Flags().GetString(flagType)
			nodeType := types.NodeTypeFromString(typeName)
			if nodeType == types.UnspecifiedNodeType {
				exitWithErrorMsgf("ERR: Invalid node type, can be either %s\n", validTargetValues)
				return
			}

			isLinux := runtime.GOOS == "linux"
			requireServiceFileForValidatorOnLinux := nodeType == types.ValidatorNode && isLinux

			serviceFilePath, _ := cmd.Flags().GetString(flagServiceFile)
			if requireServiceFileForValidatorOnLinux && serviceFilePath == "" {
				exitWithErrorMsgf("ERR: --%s is required on Linux to check validator setting\n", flagServiceFile)
				return
			} else if nodeType != types.ValidatorNode && serviceFilePath != "" {
				exitWithErrorMsgf("ERR: remove flag \"--%s\", only be used for validator on Linux\n", flagServiceFile)
				return
			}

			defer func() {
				waitGroup.Wait()
				if len(checkRecords) == 0 {
					fmt.Println("All checks passed")
					return
				}

				printCheckRecords()
				os.Exit(1)
			}()

			home := args[0]
			checkHome(home)

			checkHomeKeyring(home, nodeType == types.ValidatorNode)
			checkHomeConfig(home, nodeType)
			checkHomeData(home, nodeType)
			if requireServiceFileForValidatorOnLinux {
				checkServiceFileForValidatorOnLinux(home, serviceFilePath)
			}

			fmt.Println("NOTICE: some tasks need to be checked manually:")

			var countNotice int
			printNotice := func(message, suggest string) {
				countNotice++
				fmt.Printf("%d. %s\n", countNotice, message)
				if suggest != "" {
					fmt.Println("> " + suggest)
				}
			}
			printNotice("Ensure P2P port is open on firewall", "sudo ufw status")
			if nodeType == types.ValidatorNode {
				printNotice("Ensure RPC port is whitelisted only health-check on firewall", "sudo ufw status")
				printNotice("Ensure Rest-API, Json-RPC ports are not allowed from outside", "sudo ufw status")
			} else if nodeType == types.RpcNode || nodeType == types.ArchivalNode {
				printNotice("Ensure RPC, Rest-API, Json-RPC ports are open on firewall", "sudo ufw status")
			} else if nodeType == types.SnapshotNode {
				printNotice("Ensure RPC port is open on firewall", "sudo ufw status")
				printNotice("Ensure Rest-API, Json-RPC ports are not allowed from outside", "sudo ufw status")
			}
			printNotice("Check config.toml for 'fast_sync' and 'block_sync', if exists, set to true", "")
			fmt.Println("WARN: after checked and fixed all issues, re-check again using this tool before running node, otherwise you probably miss something")
		},
	}

	cmd.Flags().String(flagType, "", fmt.Sprintf("type of node to check, can be: %s", validTargetValues))
	cmd.Flags().String(flagServiceFile, "", "path to the service file to check, required for validator node on Linux")

	return cmd
}

func checkLatestRelease() {
	waitGroup.Add(1)
	defer func() {
		r := recover()
		if r != nil {
			printfStdErr("ERR: failed to check latest release: %v\n", r)
		}
		waitGroup.Done()
	}()

	resp, err := http.Get("https://api.github.com/repos/bcdevtools/node-setup-check/releases/latest")
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}

	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		panic(err)
	}

	latestTagName := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(constants.VERSION, "v")
	if latestTagName != currentVersion {
		warnRecord(fmt.Sprintf("latest release is v%s, must use latest version to prevent bugs and new logics", latestTagName), "")
	}
}

func init() {
	rootCmd.AddCommand(GetCheckCmd())
}
