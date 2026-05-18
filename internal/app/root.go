package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/learn0208/mailcli/internal/config"
	"github.com/learn0208/mailcli/internal/domain"
)

// Version is the CLI release version (override at link time: -ldflags "-X .../internal/app.Version=v1.2.3").
var Version = "0.1.0"

var (

	cfgFile       string
	profile       string
	verbose       bool
	endpoint      string
	user          string
	password      string
	authType      string
	userAgentFlag string
)

func defaultConfigPath() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".mailcli.yaml")
	}
	candidates := []string{
		filepath.Join(h, ".mailcli.yaml"),
		filepath.Join(h, ".ews-cli.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}

func expandUserPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "~/") {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(h, p[2:]), nil
	}
	return p, nil
}

func loadMergedProfile() (config.Profile, config.AppSettings, error) {
	path, err := expandUserPath(cfgFile)
	if err != nil {
		return config.Profile{}, config.AppSettings{}, err
	}
	p, app, err := config.Load(path, profile)
	if err != nil {
		return config.Profile{}, config.AppSettings{}, err
	}
	if endpoint != "" {
		p.Endpoint = endpoint
	}
	if user != "" {
		p.User = user
	}
	if authType != "" {
		p.AuthType = authType
	}
	config.ApplyEnv(&p)
	config.ApplyEnvAppSettings(&app)
	return p, app, nil
}

func resolveUserAgent(app config.AppSettings) string {
	if s := strings.TrimSpace(userAgentFlag); s != "" {
		return s
	}
	if s := strings.TrimSpace(app.UserAgent); s != "" {
		return s
	}
	return fmt.Sprintf("mailcli/%s", Version)
}

func resolvePassword(p config.Profile) (string, error) {
	if p.EffectiveAuth() == "oauth" {
		return "", nil
	}
	if pw := strings.TrimSpace(os.Getenv("MAILCLI_PASSWORD")); pw != "" {
		return pw, nil
	}
	if pw := strings.TrimSpace(os.Getenv("EWS_PASSWORD")); pw != "" {
		return pw, nil
	}
	if strings.TrimSpace(password) != "" {
		return password, nil
	}
	if pw := strings.TrimSpace(p.Password); pw != "" {
		return pw, nil
	}
	fmt.Fprint(os.Stderr, "Password: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	fmt.Fprintln(os.Stderr)
	return string(b), nil
}

func requireSupportedProtocol(p config.Profile) error {
	proto := domain.NormalizeProtocol(p.Protocol)
	if !proto.Supported() {
		return fmt.Errorf("protocol %q is not supported yet (available: ews)", p.Protocol)
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:           "mailcli",
	Short:         "mailCli — cross-protocol mail CLI (EWS today; IMAP/SMTP planned)",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultConfigPath(), "config file path")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "default", "profile name in config")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "log raw HTTP request/response")
	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", "", "override EWS endpoint URL (https)")
	rootCmd.PersistentFlags().StringVar(&user, "user", "", "override mailbox / username")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "password (prefer MAILCLI_PASSWORD or leave empty to prompt)")
	rootCmd.PersistentFlags().StringVar(&authType, "auth-type", "", "override auth: basic, ntlm, oauth")
	rootCmd.PersistentFlags().StringVar(&userAgentFlag, "user-agent", "", "override HTTP User-Agent (default mailcli/<version>)")

	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.AddCommand(searchCmd(), sendCmd(), discoverCmd(), showCmd())
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return err
	}
	return nil
}
