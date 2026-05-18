package app

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func discoverCmd() *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Print common EWS endpoint hints for a mailbox (no network autodiscover yet)",
		RunE: func(cmd *cobra.Command, args []string) error {
			email = strings.TrimSpace(email)
			if email == "" {
				return fmt.Errorf("--user / email is required")
			}
			at := strings.LastIndex(email, "@")
			if at < 0 {
				return fmt.Errorf("expected user@domain in %q", email)
			}
			domain := strings.ToLower(strings.TrimSpace(email[at+1:]))
			fmt.Println("Try these endpoints (verify with your admin):")
			fmt.Printf("  Exchange Online: https://outlook.office365.com/EWS/Exchange.asmx\n")
			fmt.Printf("  On-prem (example): https://mail.%s/EWS/Exchange.asmx\n", domain)
			fmt.Printf("  On-prem (example): https://ews.%s/EWS/Exchange.asmx\n", domain)
			fmt.Println("\nThen set endpoint in ~/.mailcli.yaml (or legacy ~/.ews-cli.yaml) or MAILCLI_ENDPOINT.")
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "user", "", "mailbox address (user@domain)")
	_ = cmd.MarkFlagRequired("user")
	return cmd
}
