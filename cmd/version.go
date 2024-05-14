package cmd

import (
	"fmt"
	"github.com/EscanBE/node-setup-check/constants"
	"github.com/spf13/cobra"
	"runtime"
	"runtime/debug"
)

const (
	flagLongVersion = "long"
)

func GetVersionCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Show binary version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(constants.APP_NAME)

			printLongVersion := cmd.Flag(flagLongVersion).Changed

			if printLongVersion {
				buildInfo, ok := debug.ReadBuildInfo()
				if ok {
					fmt.Println("Build dependencies:")
				}
				for _, dep := range buildInfo.Deps {
					if dep.Replace != nil {
						fmt.Printf("- %s@%s => %s@%s\n", dep.Path, dep.Version, dep.Replace.Path, dep.Replace.Version)
					} else {
						fmt.Printf("- %s@%s\n", dep.Path, dep.Version)
					}
				}
			}

			fmt.Printf("%-11s %s\n", "Version:", constants.VERSION)
			fmt.Printf("%-11s %s\n", "Commit:", constants.COMMIT_HASH)
			fmt.Printf("%-11s %s\n", "Build date:", constants.BUILD_DATE)

			if printLongVersion {
				fmt.Printf("%-11s %s %s/%s\n", "Go:", runtime.Version(), runtime.GOOS, runtime.GOARCH)
			}
		},
	}

	cmd.PersistentFlags().Bool(
		flagLongVersion, false, "print extra version information",
	)

	return cmd
}

func init() {
	rootCmd.AddCommand(GetVersionCmd())
}
