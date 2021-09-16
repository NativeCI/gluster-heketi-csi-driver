package cmd

import (
	"github.com/NativeCI/gluster-heketi-csi-driver/glusterfs"
	"github.com/spf13/cobra"
)

var nodeserverCmd = &cobra.Command{
	Use:   "nodeserver",
	Short: "Run nodeserver",
	Run: func(cmd *cobra.Command, args []string) {
		driver := glusterfs.New(rootConfig)
		server := glusterfs.NewNodeServer(driver)
		server.Run()
	},
}

func init() {
	rootCmd.AddCommand(nodeserverCmd)
}
