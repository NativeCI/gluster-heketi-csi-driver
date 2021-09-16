package cmd

import (
	"github.com/NativeCI/gluster-heketi-csi-driver/glusterfs"
	"github.com/spf13/cobra"
)

var controllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "Run controller",
	Run: func(cmd *cobra.Command, args []string) {
		driver := glusterfs.New(rootConfig)
		server := glusterfs.NewControllerServer(driver)
		server.Run()
	},
}

func init() {
	rootCmd.AddCommand(controllerCmd)
}
