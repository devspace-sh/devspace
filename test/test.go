package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	rootCmd := &cobra.Command{
		Use: "sdfssfd",
		Short: "Welcome to the DevSpace!",
		Long: `Example bois`,
	}

	testCmd := &cobra.Command{
		Use: "test",
		Short: "Welcome to the DevSpace!",
		Long: `Example bois`,
	}

	testCmd.AddCommand(&cobra.Command{
		Use: "abc",
		Short: "Welcome to the DevSpace!",
		Long: `Example bois`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("abc v2 %v", args)
			return nil
		},
	})
	testCmd.AddCommand(&cobra.Command{
		Use: "def",
		Short: "Welcome to the DevSpace!",
		Long: `Example bois`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("def %v", args)
			return nil
		},
	})

	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "sutest",
		Short: "Welcome to the DevSpace!",
		Long:  `Example bois`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("sutest %v", args)
			return nil
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "var",
		Short: "Welcome to the DevSpace!",
		Long:  `Example bois`,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, v := range os.Environ() {
				fmt.Println(v)
			}
			return nil
		},
	})

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
	}
}