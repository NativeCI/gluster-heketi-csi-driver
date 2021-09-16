package cmd

import (
	"github.com/NativeCI/gluster-heketi-csi-driver/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gluster-heketi-csi",
	Short: "Gluster heketi csi plugin",
}

var rootConfig = config.NewConfig()

func init() {
	rootCmd.PersistentFlags().StringVar(&rootConfig.NodeID, "nodeid", "", "CSI node id")
	_ = rootCmd.MarkPersistentFlagRequired("nodeid")

	rootCmd.PersistentFlags().StringVar(&rootConfig.Endpoint, "endpoint", "", "CSI endpoint")

	rootCmd.PersistentFlags().StringVar(&rootConfig.HeketiURL, "heketiurl", "", "heketi rest endpoint")

	rootCmd.PersistentFlags().StringVar(&rootConfig.HeketiUser, "username", "", "heketi user name")

	rootCmd.PersistentFlags().StringVar(&rootConfig.HeketiSecret, "heketisecret", "", "heketi rest user secret")
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
