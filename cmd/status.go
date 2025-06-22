package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status of the KastQl Engine",
	Long:  `Status of the KastQl Engine`,
	Run: func(cmd *cobra.Command, args []string) {
		Status()
	},
}

func Status() {
	fmt.Printf("Current KastQl status: %s \n", "Working")
}

func init() {
	RootCmd.AddCommand(StatusCmd)
}
