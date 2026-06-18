package display

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/samjoshuadud/nixs/internal/api"
)

var (
	nameStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))  // cyan-ish
	versionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))              // green
	descStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))              // light gray
	labelStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33")) // yellow
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))              // dark gray
	optNameStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("135")) // purple
	typeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))              // amber
)

// PrintPackageList prints results like pacman -Ss
func PrintPackageList(packages []api.Package) {
	for _, p := range packages {
		// nixpkgs/firefox 124.0.1
		line := fmt.Sprintf("%s/%s %s",
			dimStyle.Render("nixpkgs"),
			nameStyle.Render(p.Name),
			versionStyle.Render(p.Version),
		)
		fmt.Println(line)

		// description
		desc := p.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Println("    " + descStyle.Render(desc))

		// programs provided
		if len(p.Programs) > 0 {
			fmt.Println("    " + dimStyle.Render("programs: "+strings.Join(p.Programs, ", ")))
		}

		fmt.Println()
	}

	fmt.Printf(dimStyle.Render("  %d result(s)\n"), len(packages))
}

// PrintPackageInfo prints detailed info like pacman -Si
func PrintPackageInfo(p api.Package) {
	printField("Name", p.Name)
	printField("Version", p.Version)

	if len(p.Homepage) > 0 {
		printField("Homepage", p.Homepage[0])
	}

	if len(p.License) > 0 {
		printField("License", p.License[0].FullName)
	}

	if len(p.Maintainers) > 0 {
		names := make([]string, 0, len(p.Maintainers))
		for _, m := range p.Maintainers {
			names = append(names, m.Name)
		}
		printField("Maintainers", strings.Join(names, ", "))
	}

	if len(p.Programs) > 0 {
		printField("Programs", strings.Join(p.Programs, ", "))
	}

	desc := p.Description
	if p.LongDesc != "" {
		desc = p.LongDesc
	}
	printField("Description", desc)

	// Show how to install — this is what nix-search-cli doesn't do
	fmt.Println()
	fmt.Println(labelStyle.Render("  Install:"))
	fmt.Println(dimStyle.Render("    # configuration.nix"))
	fmt.Printf("    environment.systemPackages = [ pkgs.%s ];\n", p.Name)
	fmt.Println()
	fmt.Println(dimStyle.Render("    # nix profile (flakes)"))
	fmt.Printf("    nix profile install nixpkgs#%s\n", p.Name)
}

// PrintOptionList prints NixOS or Home Manager options
func PrintOptionList(options []api.Option, source string) {
	for _, o := range options {
		// source/option.name
		line := fmt.Sprintf("%s/%s",
			dimStyle.Render(strings.ToLower(source)),
			optNameStyle.Render(o.Name),
		)
		fmt.Println(line)

		if o.Type != "" {
			fmt.Println("    " + typeStyle.Render("type: "+o.Type))
		}

		desc := o.Description
		if desc == "" {
			desc = "(no description)"
		}
		// strip HTML tags that sometimes appear in descriptions
		desc = stripHTML(desc)
		fmt.Println("    " + descStyle.Render(desc))

		if o.Default != "" && o.Default != "null" {
			fmt.Println("    " + dimStyle.Render("default: "+o.Default))
		}

		if o.Example != "" {
			fmt.Println("    " + dimStyle.Render("example: "+o.Example))
		}

		fmt.Println()
	}

	fmt.Printf(dimStyle.Render("  %d result(s) from %s\n"), len(options), source)
}

func printField(label, value string) {
	padded := fmt.Sprintf("%-14s", label)
	fmt.Printf("%s: %s\n", labelStyle.Render(padded), value)
}

func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}
