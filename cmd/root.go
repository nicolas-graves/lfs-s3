package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"git.sr.ht/~ngraves/lfs-s3/service"
)

var (
	printVersion bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd *cobra.Command

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd = &cobra.Command{
		Use:   "git-lfs-s3",
		Short: "git-lfs custom transfer adapter to store all data in a S3 container",
		Long: `git-lfs-s3 treats a S3 bucket as the remote store for LFS object
		data.`,
		Run: rootCommand,
	}

	RootCmd.Flags().BoolVarP(&printVersion, "version", "", false, "Print version")
	RootCmd.SetUsageFunc(usageCommand)

}

func usageCommand(cmd *cobra.Command) error {
	usage := `
Usage:
  git-lfs-s3 [options]

Options:
  --version    Report the version number and exit

Note:
  This tool should only be called by git-lfs as documented in Custom Transfers:
  https://github.com/git-lfs/git-lfs/blob/master/docs/custom-transfers.md

  The arguments should be provided via gitconfig at lfs.customtransfer.<name>.args
`
	fmt.Fprintf(os.Stderr, usage)
	return nil
}

func rootCommand(cmd *cobra.Command, args []string) {

	if printVersion {
		os.Stderr.WriteString(fmt.Sprintf("git-lfs-s3 %v\n", Version))
		os.Exit(0)
	}

	service.Serve(os.Stdin, os.Stdout, os.Stderr)
}
