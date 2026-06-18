package cli

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"github.com/samjoshuadud/nixs/internal/api"
	"github.com/samjoshuadud/nixs/internal/display"
)

var (
	channel    string
	showInfo   bool
	searchAll  bool
	searchHM   bool
	searchOpts bool
	stable     bool
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

		if stable {
			channel = "stable"
		}

		if searchHM {
			return runHMSearch(query)
		}
		if searchOpts {
			return runOptsSearch(query)
		}

		if !searchAll {
			return runPackageSearch(query, showInfo)
		}

		// If --all is provided, search everything concurrently
		ch := resolveChannel(channel)
		
		var wg sync.WaitGroup
		var pkgRes []api.Package
		var optsRes []api.Option
		var hmRes []api.Option
		var pkgErr, optsErr, hmErr error

		wg.Add(3)
		go func() {
			defer wg.Done()
			pkgRes, pkgErr = api.SearchPackages(query, ch, maxResults)
		}()
		go func() {
			defer wg.Done()
			optsRes, optsErr = api.SearchOptions(query, ch, maxResults)
		}()
		go func() {
			defer wg.Done()
			hmRes, hmErr = api.SearchHomeManager(query, maxResults)
		}()

		wg.Wait()

		// Print Packages
		if pkgErr != nil {
			fmt.Printf("packages search error: %v\n", pkgErr)
		} else if len(pkgRes) == 0 {
			fmt.Printf("no packages found for '%s'\n\n", query)
		} else if showInfo {
			display.PrintPackageInfo(pkgRes[0])
		} else {
			display.PrintPackageList(pkgRes)
		}

		// Print NixOS Options
		if optsErr != nil {
			fmt.Printf("options search error: %v\n", optsErr)
		} else if len(optsRes) == 0 {
			fmt.Printf("no NixOS options found for '%s'\n\n", query)
		} else {
			display.PrintOptionList(optsRes, "NixOS")
		}

		// Print Home Manager Options
		if hmErr != nil {
			fmt.Printf("home manager search error: %v\n", hmErr)
		} else if len(hmRes) == 0 {
			fmt.Printf("no Home Manager options found for '%s'\n\n", query)
		} else {
			display.PrintOptionList(hmRes, "Home Manager")
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
	rootCmd.Flags().BoolVar(&stable, "stable", false, "search stable channel (alias for -c stable)")
	rootCmd.Flags().BoolVarP(&showInfo, "info", "i", false, "show full package details")
	rootCmd.Flags().BoolVar(&searchAll, "all", false, "search all ecosystems")
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
	if c == "stable" {
		return api.GetLatestStableChannel()
	}
	return c
}
