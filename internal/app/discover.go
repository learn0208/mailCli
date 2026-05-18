package app

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/learn0208/mailcli/internal/config"
	"github.com/learn0208/mailcli/internal/domain"
)

func discoverCmd() *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Print endpoint or IMAP/SMTP hints for a mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			email = strings.TrimSpace(email)
			if email == "" {
				return fmt.Errorf("--user / email is required")
			}
			at := strings.LastIndex(email, "@")
			if at < 0 {
				return fmt.Errorf("expected user@domain in %q", email)
			}
			domainName := strings.ToLower(strings.TrimSpace(email[at+1:]))

			p, _, err := loadMergedProfile()
			if err != nil {
				return err
			}
			if domain.NormalizeProtocol(p.Protocol) == domain.ProtocolIMAP {
				printIMAPHints(email, domainName)
				return nil
			}
			if prov := config.LookupProviderByDomain(domainName); prov != nil && strings.TrimSpace(p.Endpoint) == "" {
				printIMAPHints(email, domainName)
				return nil
			}
			printEWSHints(domainName)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "user", "", "mailbox address (user@domain)")
	_ = cmd.MarkFlagRequired("user")
	return cmd
}

func printEWSHints(domainName string) {
	fmt.Println("Try these EWS endpoints (verify with your admin):")
	fmt.Printf("  Exchange Online: https://outlook.office365.com/EWS/Exchange.asmx\n")
	fmt.Printf("  On-prem (example): https://mail.%s/EWS/Exchange.asmx\n", domainName)
	fmt.Printf("  On-prem (example): https://ews.%s/EWS/Exchange.asmx\n", domainName)
	fmt.Println("\nSet endpoint in ~/.mailcli.yaml or MAILCLI_ENDPOINT.")
}

func printIMAPHints(email, domainName string) {
	if prov := config.LookupProviderByDomain(domainName); prov != nil {
		fmt.Printf("Known provider: %s (%s)\n\n", prov.DisplayName, prov.ID)
		fmt.Printf("  IMAP: %s\n", prov.IMAPHost)
		fmt.Printf("  SMTP: %s\n", prov.SMTPHost)
		fmt.Println()
		fmt.Println("Authentication:")
		fmt.Println(prov.AuthHint)
		fmt.Println()
		fmt.Println("Example ~/.mailcli.yaml profile:")
		fmt.Println(config.FormatProviderYAML(*prov, email))
		fmt.Println("Commands: mailcli providers show", prov.ID, "--user", email)
		return
	}
	fmt.Println("Common IMAP/SMTP hosts (verify with your provider):")
	fmt.Printf("  IMAP: imap.%s:993 (TLS)\n", domainName)
	fmt.Printf("  SMTP: smtp.%s:587 (STARTTLS) or :465 (TLS)\n", domainName)
	fmt.Println("\nBuilt-in presets: mailcli providers list")
	fmt.Println("Set MAILCLI_PASSWORD (often an app/authorization code, not your web login password).")
}
