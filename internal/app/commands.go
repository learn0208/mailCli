package app

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/learn0208/mailcli/internal/config"
	"github.com/learn0208/mailcli/internal/domain"
	"github.com/learn0208/mailcli/internal/protocol/ews/search"
	"github.com/learn0208/mailcli/internal/protocol/ews/send"
	"github.com/learn0208/mailcli/internal/protocol/ews/show"
	imapsearch "github.com/learn0208/mailcli/internal/protocol/imap/search"
	imapshow "github.com/learn0208/mailcli/internal/protocol/imap/show"
	smtpsend "github.com/learn0208/mailcli/internal/protocol/smtp/send"
)

func searchCmd() *cobra.Command {
	var (
		subject, query, body, from, to string
		since, until            string
		folder                  string
		unread, read            bool
		attachments             bool
		limit                   int
		defaultDays             int
		noDefaultDateWindow     bool
		output                  string
	)
	cmd := &cobra.Command{
		Use:   "search [keyword]",
		Short: "Search messages in a folder",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && query == "" && subject == "" && from == "" && body == "" {
				query = args[0]
			}
			p, app, err := loadMergedProfile()
			if err != nil {
				return err
			}
			if err := requireSupportedProtocol(p); err != nil {
				return err
			}
			common := searchOptsCommon{
				Subject: subject, Query: query, Body: body, From: from, To: to,
				Since: since, Until: until, Folder: folder,
				Unread: unread, Read: read, Attachments: attachments,
				Limit: limit, DefaultDays: defaultDays, Output: output,
				UserSetSince:        cmd.Flags().Changed("since"),
				UserSetUntil:        cmd.Flags().Changed("until"),
				NoDefaultDateWindow: noDefaultDateWindow,
			}
			switch domain.NormalizeProtocol(p.Protocol) {
			case domain.ProtocolIMAP:
				return runIMAPSearch(p, imapsearch.Options{
					Subject: common.Subject, Query: common.Query, Body: common.Body, From: common.From, To: common.To,
					Since: common.Since, Until: common.Until, Folder: common.Folder,
					Unread: common.Unread, Read: common.Read, Attachment: common.Attachments,
					Limit: common.Limit, Output: common.Output,
					UserSetSince: common.UserSetSince, UserSetUntil: common.UserSetUntil,
					NoDefaultDateWindow: common.NoDefaultDateWindow, DefaultDays: common.DefaultDays,
					Verbose: verbose,
				})
			default:
				return runEWSSearch(p, app, search.Options{
					Subject: common.Subject, Body: common.Body, From: common.From, To: common.To,
					Since: common.Since, Until: common.Until, Folder: common.Folder,
					Unread: common.Unread, Read: common.Read, Attachment: common.Attachments,
					Limit: common.Limit, Output: common.Output, Verbose: verbose,
					UserSetSince: common.UserSetSince, UserSetUntil: common.UserSetUntil,
					NoDefaultDateWindow: common.NoDefaultDateWindow, DefaultDays: common.DefaultDays,
				})
			}
		},
	}
	cmd.Flags().StringVar(&subject, "subject", "", "subject contains (case-insensitive)")
	cmd.Flags().StringVar(&query, "query", "", "keyword in subject, from display name, or body (IMAP)")
	cmd.Flags().StringVar(&body, "body", "", "body text contains")
	cmd.Flags().StringVar(&from, "from", "", "from address contains")
	cmd.Flags().StringVar(&to, "to", "", "to recipients contains")
	cmd.Flags().StringVar(&since, "since", "", "received on/after (RFC3339, date, or e.g. \"7 days ago\"); omit with default window: last N days")
	cmd.Flags().StringVar(&until, "until", "", "received on/before")
	cmd.Flags().StringVar(&folder, "folder", "Inbox", "folder name (default Inbox)")
	cmd.Flags().BoolVar(&unread, "unread", false, "only unread")
	cmd.Flags().BoolVar(&read, "read", false, "only read")
	cmd.Flags().BoolVar(&attachments, "attachments", false, "only messages with attachments")
	cmd.Flags().IntVar(&limit, "limit", 10, "max messages (newest first within the query window)")
	cmd.Flags().IntVar(&defaultDays, "default-days", 30, "when neither --since nor --until is given, restrict to the last N calendar days (unless --no-default-date-window)")
	cmd.Flags().BoolVar(&noDefaultDateWindow, "no-default-date-window", false, "search all time when --since/--until are omitted (does not apply if you pass either flag)")
	cmd.Flags().StringVar(&output, "output", "table", "output: table or json")
	return cmd
}

func showCmd() *cobra.Command {
	var itemID, changeKey, format, folder string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show one message body and headers",
		Long:  "Load a message by ItemId/UID from a previous search. Use --format json for machine-readable output.",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, app, err := loadMergedProfile()
			if err != nil {
				return err
			}
			if err := requireSupportedProtocol(p); err != nil {
				return err
			}
			switch domain.NormalizeProtocol(p.Protocol) {
			case domain.ProtocolIMAP:
				return runIMAPShow(p, folder, imapshow.Options{
					ItemID: itemID, ChangeKey: changeKey, Format: format,
				})
			default:
				return runEWSShow(p, app, show.Options{
					ItemID: itemID, ChangeKey: changeKey, Format: format,
				})
			}
		},
	}
	cmd.Flags().StringVar(&itemID, "item-id", "", "message id (EWS ItemId or IMAP UID from search)")
	cmd.Flags().StringVar(&changeKey, "change-key", "", "optional EWS ChangeKey from search")
	cmd.Flags().StringVar(&folder, "folder", "Inbox", "folder name (IMAP; must match search)")
	cmd.Flags().StringVar(&format, "format", "text", "text | html | json")
	_ = cmd.MarkFlagRequired("item-id")
	return cmd
}

func sendCmd() *cobra.Command {
	var (
		toStr, ccStr, bccStr string
		subject              string
		textBody, htmlBody   string
		attach               []string
		importance           string
		fromAddr             string
		fromName             string
		fromNamePlainUTF8    bool
		fromAddressOnly      bool
		jsonOut              bool
		noVerifySent         bool
		verifySentWaitSec    int
	)
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send an email message",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, app, err := loadMergedProfile()
			if err != nil {
				return err
			}
			if err := requireSupportedProtocol(p); err != nil {
				return err
			}
			fromSMTP := strings.TrimSpace(fromAddr)
			if fromSMTP == "" {
				fromSMTP = config.InferSMTPAddress(p)
			}
			fromDisp := strings.TrimSpace(fromName)
			if fromDisp == "" {
				fromDisp = strings.TrimSpace(p.DisplayName)
			}
			addrOnly := fromAddressOnly || p.FromAddressOnly
			if addrOnly {
				fromDisp = ""
			}
			toAddrs := splitAddresses(toStr)

			switch domain.NormalizeProtocol(p.Protocol) {
			case domain.ProtocolIMAP:
				return runSMTPSend(p, smtpsend.Options{
					From: fromSMTP, FromDisplayName: fromDisp,
					To: toAddrs, Cc: splitAddresses(ccStr), Bcc: splitAddresses(bccStr),
					Subject: subject, TextBody: textBody, HTMLBody: htmlBody, Attach: attach,
				}, noVerifySent, verifySentWaitSec, jsonOut)
			default:
				imp, err := send.NormalizeImportance(importance)
				if err != nil {
					return err
				}
				return runEWSSend(p, app, send.Options{
					From:                     fromSMTP,
					FromDisplayName:          fromDisp,
					FromDisplayNamePlainUTF8: fromNamePlainUTF8 || p.FromDisplayNamePlainUTF8,
					FromAddressOnly:          addrOnly,
					To:                       toAddrs,
					Cc:                       send.SplitAddresses(ccStr),
					Bcc:                      send.SplitAddresses(bccStr),
					Subject:                  subject,
					TextBody:                 textBody,
					HTMLBody:                 htmlBody,
					Attach:                   attach,
					Importance:               imp,
				}, noVerifySent, verifySentWaitSec, jsonOut)
			}
		},
	}
	cmd.Flags().StringVar(&toStr, "to", "", "recipients (comma-separated)")
	cmd.Flags().StringVar(&ccStr, "cc", "", "cc recipients")
	cmd.Flags().StringVar(&bccStr, "bcc", "", "bcc recipients")
	cmd.Flags().StringVar(&subject, "subject", "", "subject line")
	cmd.Flags().StringVar(&textBody, "text", "", "plain text body")
	cmd.Flags().StringVar(&htmlBody, "html", "", "HTML body")
	cmd.Flags().StringSliceVar(&attach, "attach", nil, "attachment file path (repeatable)")
	cmd.Flags().StringVar(&importance, "importance", "", "high | normal | low (EWS only)")
	cmd.Flags().StringVar(&fromAddr, "from", "", "sender address (default: authenticated user)")
	cmd.Flags().StringVar(&fromName, "from-name", "", "UTF-8 sender display name")
	cmd.Flags().BoolVar(&fromNamePlainUTF8, "from-name-plain-utf8", false, "EWS: put raw UTF-8 in Name")
	cmd.Flags().BoolVar(&fromAddressOnly, "from-address-only", false, "EWS: ASCII From with Name=Email")
	cmd.Flags().BoolVar(&noVerifySent, "no-verify-sent", false, "do not search Sent folder after send")
	cmd.Flags().IntVar(&verifySentWaitSec, "verify-sent-wait", 5, "seconds to wait before checking Sent folder (0=no wait; max 120)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print result as JSON")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("subject")
	return cmd
}

type searchOptsCommon struct {
	Subject, Query, Body, From, To, Since, Until, Folder, Output string
	Unread, Read, Attachments                               bool
	Limit, DefaultDays                                      int
	UserSetSince, UserSetUntil, NoDefaultDateWindow         bool
}
