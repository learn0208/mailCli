package app

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/learn0208/mailcli/internal/config"
)

func profileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Inspect merged profile settings",
	}
	cmd.AddCommand(profileShowCmd())
	return cmd
}

func profileShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print effective profile after provider presets and env",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, _, err := loadMergedProfile()
			if err != nil {
				return err
			}
			prov := config.LookupProvider(p)
			out := struct {
				Protocol string `json:"protocol"`
				Provider string `json:"provider,omitempty"`
				User     string `json:"user"`
				IMAP     struct {
					Host string `json:"host"`
					TLS  bool   `json:"tls"`
				} `json:"imap"`
				SMTP struct {
					Host      string `json:"host"`
					TLS       bool   `json:"tls"`
					StartTLS  bool   `json:"starttls"`
				} `json:"smtp"`
				PresetApplied string `json:"preset_applied,omitempty"`
			}{
				Protocol: p.Protocol,
				Provider: p.Provider,
				User:     p.User,
			}
			out.IMAP.Host = p.IMAP.Host
			out.IMAP.TLS = config.BoolDefault(p.IMAP.TLS, true)
			out.SMTP.Host = p.SMTP.Host
			out.SMTP.TLS = config.BoolDefault(p.SMTP.TLS, false)
			out.SMTP.StartTLS = config.BoolDefault(p.SMTP.StartTLS, false)
			if prov != nil {
				out.PresetApplied = prov.ID
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(out); err != nil {
				return err
			}
			if out.IMAP.Host == "" {
				fmt.Fprintln(os.Stderr, "提示: imap.host 为空，请设置 provider 或 imap.host（mailcli providers list）")
			}
			return nil
		},
	}
}
