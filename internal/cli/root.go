package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/samjoshuadud/nixs/internal/api"
	"github.com/samjoshuadud/nixs/internal/display"
)

var (
	channel    string
	showInfo   bool
	searchPkg  bool
	searchHM   bool
	searchOpts bool
	maxResults int
)

var rootCmd = &cobra.Command{
	Use:   "nixs <query>",
	Short: "A fast, unified Nix package and option search",
	Long:  `nixs — search nixpkgs, NixOS options, and Home Manager options from one place.`,
	Example: `  nixs firefox              # search packages
  nixs -i firefox           # show full package info
  nixs --hm neovim          # search Home Manager options
  nixs --opt services.nginx # search NixOS options
  nixs --stable firefox     # search stable channel`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		if searchPkg {
			return runPackageSearch(query, showInfo)
		}
		if searchHM {
			return runHMSearch(query)
		}
		if searchOpts {
			return runOptsSearch(query)
		}

		// If no specific flag is provided, search everything
		errPkg := runPackageSearch(query, showInfo)
		if errPkg != nil {
			fmt.Printf("packages search error: %v\n", errPkg)
		}
		errOpts := runOptsSearch(query)
		if errOpts != nil {
			fmt.Printf("options search error: %v\n", errOpts)
		}
		errHM := runHMSearch(query)
		if errHM != nil {
			fmt.Printf("home manager search error: %v\n", errHM)
		}

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&channel, "channel", "c", "unstable", "nixpkgs channel (unstable, stable)")
	rootCmd.Flags().BoolVarP(&showInfo, "info", "i", false, "show full package details")
	rootCmd.Flags().BoolVar(&searchPkg, "pkg", false, "search packages only")
	rootCmd.Flags().BoolVar(&searchHM, "hm", false, "search Home Manager options only")
	rootCmd.Flags().BoolVar(&searchOpts, "opt", false, "search NixOS options only")
	rootCmd.Flags().IntVarP(&maxResults, "max", "m", 20, "max results to show")
}

func runPackageSearch(query string, info bool) error {
	ch := resolveChannel(channel)
	results, err := api.SearchPackages(query, ch, maxResults)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	if len(results) == 0 {
		fmt.Printf("no packages found for '%s'\n\n", query)
		return nil
	}
	if info {
		display.PrintPackageInfo(results[0])
	} else {
		display.PrintPackageList(results)
	}
	return nil
}

func runHMSearch(query string) error {
	results, err := api.SearchHomeManager(query, maxResults)
	if err != nil {
		return fmt.Errorf("home manager search failed: %w", err)
	}
	if len(results) == 0 {
		fmt.Printf("no Home Manager options found for '%s'\n\n", query)
		return nil
	}
	display.PrintOptionList(results, "Home Manager")
	return nil
}

func runOptsSearch(query string) error {
	ch := resolveChannel(channel)
	results, err := api.SearchOptions(query, ch, maxResults)
	if err != nil {
		return fmt.Errorf("options search failed: %w", err)
	}
	if len(results) == 0 {
		fmt.Printf("no NixOS options found for '%s'\n\n", query)
		return nil
	}
	display.PrintOptionList(results, "NixOS")
	return nil
}

func resolveChannel(c string) string {
	switch c {
	case "stable", "24.11":
		return "24.11"
	default:
		return "unstable"
	}
}
