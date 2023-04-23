package cmd

import (
	"flag"
	"fmt"
	"os"

	"git.sr.ht/~ngraves/lfs-s3/service"
)

var (
	printVersion bool
)

func init() {
	flag.BoolVar(&printVersion, "version", false, "Print version")

	flag.Usage = func() {
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
	}
}

// Execute runs the main logic of the program and handles command line arguments.
func Execute() {
	flag.Parse()

	if printVersion {
		os.Stderr.WriteString(fmt.Sprintf("git-lfs-s3 %v\n", Version))
		os.Exit(0)
	}

	service.Serve(os.Stdin, os.Stdout, os.Stderr)
}
