package display

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"

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
		desc := cleanText(stripHTML(p.Description))
		if desc == "" {
			desc = "(no description)"
		}
		desc = strings.ReplaceAll(desc, "\n", "\n    ")
		fmt.Println("    " + descStyle.Render(desc))

		// programs provided
		if len(p.Programs) > 0 {
			fmt.Println("    " + dimStyle.Render("programs: "+strings.Join(p.Programs, ", ")))
		}
	}

	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  → %d results", len(packages))))
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
	desc = cleanText(stripHTML(desc))
	if desc == "" {
		desc = "(no description)"
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

		desc := cleanText(stripHTML(o.Description))
		if desc == "" {
			desc = "(no description)"
		}
		desc = strings.ReplaceAll(desc, "\n", "\n    ")
		fmt.Println("    " + descStyle.Render(desc))

		if o.Default != "" && o.Default != "null" {
			def := cleanCode(o.Default)
			def = strings.ReplaceAll(def, "\n", "\n             ")
			fmt.Println("    " + dimStyle.Render("default: "+def))
		}

		if o.Example != "" {
			ex := cleanCode(o.Example)
			ex = strings.ReplaceAll(ex, "\n", "\n             ")
			fmt.Println("    " + dimStyle.Render("example: "+ex))
		}
	}

	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("  → %d results from %s", len(options), source)))
}

func printField(label, value string) {
	padded := fmt.Sprintf("%-14s", label)
	value = strings.ReplaceAll(value, "\n", "\n                ")
	fmt.Printf("%s: %s\n", labelStyle.Render(padded), value)
}

var roleRegex = regexp.MustCompile(`\{(file|command|env|option|manpage)\}\x60([^\x60]*)\x60`)

// cleanText trims trailing whitespace, drops leading/trailing blank lines,
// and re-flows soft-wrapped prose paragraphs from the API's fixed-column output.
func cleanText(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}

	// Re-flow: join continuation lines (non-blank lines that have no leading
	// whitespace and are not the first line) back onto the previous line.
	var reflowed []string
	for i, l := range lines {
		if i == 0 {
			reflowed = append(reflowed, l)
			continue
		}
		prev := reflowed[len(reflowed)-1]
		isList := strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "* ")
		// A continuation line: non-empty, not a list item,
		// and the previous line was also non-empty.
		if l != "" && !isList && prev != "" {
			reflowed[len(reflowed)-1] = prev + " " + l
		} else {
			reflowed = append(reflowed, l)
		}
	}
	lines = reflowed

	// Drop leading blank lines.
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	// Drop trailing blank lines.
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// cleanCode trims trailing whitespace and blank edges but preserves
// internal structure (used for default/example code blocks).
func cleanCode(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRightFunc(l, unicode.IsSpace)
	}
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "<li>", "\n- ")
	s = strings.ReplaceAll(s, "<p>", "\n")
	s = strings.ReplaceAll(s, "</p>", "\n")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")

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
	s = result.String()
	s = html.UnescapeString(s)
	s = roleRegex.ReplaceAllString(s, "$2")
	
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(s)
}
