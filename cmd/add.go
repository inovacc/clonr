package cmd

import (
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Adds a directory to the list of repositories.",
	Long:  `Adds a specified directory path to a list of repositories. The command will ask for confirmation before adding.`,
	Args:  cobra.ExactArgs(1),
	// Run: func(cmd *cobra.Command, args []string) {
	// 	path := args[0]
	// 	fullPath, err := getFullPath(path)
	// 	if err != nil {
	// 		cmd.Printf("Error getting full path: %v\n", err)
	// 		return
	// 	}
	//
	// 	cmd.Printf("Do you want to add the directory '%s' to your repositories? (y/n): ", fullPath)
	// 	var response string
	// 	_, _ = fmt.Scanln(&response)
	//
	// 	if response == "y" || response == "Y" {
	// 		cmd.Printf("Adding directory: %s\n", fullPath)
	// 		// Call a function here to save or process the path
	// 	} else {
	// 		cmd.Println("Add operation cancelled.")
	// 	}
	// },
}

func init() {
	rootCmd.AddCommand(addCmd)
}

// add a directory to the list of repositories and ask before
