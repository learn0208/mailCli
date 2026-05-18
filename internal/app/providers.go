package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	embeddocs "github.com/learn0208/mailcli/docs"
	"github.com/learn0208/mailcli/internal/config"
)

func providersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "List built-in IMAP/SMTP presets for common mail services",
	}
	cmd.AddCommand(providersListCmd(), providersShowCmd(), providersDocCmd())
	return cmd
}

func providersDocCmd() *cobra.Command {
	var listAll bool
	cmd := &cobra.Command{
		Use:   "doc [provider-id]",
		Short: "Print full setup guide for a provider (embedded documentation)",
		Long:  "Reads docs/providers/<id>.md bundled with mailcli. Use `mailcli providers doc` without id to list available guides.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if listAll || len(args) == 0 {
				return printProviderDocIndex()
			}
			id := strings.ToLower(strings.TrimSpace(args[0]))
			if config.LookupProviderByID(id) == nil {
				return fmt.Errorf("unknown provider %q (try: mailcli providers doc)", id)
			}
			b, err := fs.ReadFile(embeddocs.ProvidersFS, "providers/"+id+".md")
			if err != nil {
				return fmt.Errorf("documentation for %q not found: %w", id, err)
			}
			_, err = os.Stdout.Write(b)
			if len(b) > 0 && b[len(b)-1] != '\n' {
				_, _ = os.Stdout.WriteString("\n")
			}
			fmt.Fprintf(os.Stderr, "\nExample YAML: docs/examples/providers/%s.yaml\n", id)
			return err
		},
	}
	cmd.Flags().BoolVar(&listAll, "list", false, "list available provider documentation ids")
	return cmd
}

func printProviderDocIndex() error {
	entries, err := fs.ReadDir(embeddocs.ProvidersFS, "providers")
	if err != nil {
		return err
	}
	fmt.Println("Provider setup guides (mailcli providers doc <id>):")
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tEXAMPLE CONFIG")
	for _, p := range config.AllProviders() {
		_, _ = fmt.Fprintf(w, "%s\tdocs/examples/providers/%s.yaml\n", p.ID, p.ID)
	}
	_ = w.Flush()
	fmt.Println()
	fmt.Println("Embedded markdown files:")
	for _, e := range entries {
		if e.IsDir() || e.Name() == "README.md" {
			continue
		}
		id := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		fmt.Printf("  %s\n", id)
	}
	return nil
}

func providersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List supported provider ids",
		RunE: func(cmd *cobra.Command, args []string) error {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tDOMAINS")
			for _, p := range config.AllProviders() {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", p.ID, p.DisplayName, strings.Join(p.Domains, ", "))
			}
			return w.Flush()
		},
	}
}

func providersShowCmd() *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:   "show <provider-id>",
		Short: "Show IMAP/SMTP hosts and auth notes for a provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prov := config.LookupProviderByID(args[0])
			if prov == nil {
				return fmt.Errorf("unknown provider %q (try: mailcli providers list)", args[0])
			}
			fmt.Printf("Name:     %s (%s)\n", prov.DisplayName, prov.ID)
			fmt.Printf("Domains:  %s\n", strings.Join(prov.Domains, ", "))
			fmt.Printf("IMAP:     %s\n", prov.IMAPHost)
			fmt.Printf("SMTP:     %s\n", prov.SMTPHost)
			if len(prov.SentFolders) > 0 {
				fmt.Printf("Sent:     %s\n", strings.Join(prov.SentFolders, ", "))
			}
			fmt.Println()
			fmt.Println("Authentication:")
			fmt.Println(prov.AuthHint)
			if email = strings.TrimSpace(email); email != "" {
				fmt.Println()
				fmt.Println("Example profile:")
				fmt.Println(config.FormatProviderYAML(*prov, email))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "user", "", "example mailbox for YAML snippet")
	return cmd
}
