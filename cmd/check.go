package cmd

import (
	"fmt"
	"github.com/bcdevtools/node-setup-check/constants"
	"github.com/bcdevtools/node-setup-check/types"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"time"
)

const (
	flagType = "type"
)

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
			time.Sleep(2 * time.Second)

			typeName, _ := cmd.Flags().GetString(flagType)
			nodeType := types.NodeTypeFromString(typeName)
			if nodeType == types.UnspecifiedNodeType {
				exitWithErrorMsgf("ERR: Invalid node type, can be either %s\n", validTargetValues)
			}

			defer func() {
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
		},
	}

	cmd.Flags().String(flagType, "", fmt.Sprintf("type of node to check, can be: %s", validTargetValues))

	return cmd
}

func init() {
	rootCmd.AddCommand(GetCheckCmd())
}
