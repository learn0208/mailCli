package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/learn0208/mailcli/internal/config"
	pews "github.com/learn0208/mailcli/internal/protocol/ews"
	"github.com/learn0208/mailcli/internal/protocol/ews/search"
	"github.com/learn0208/mailcli/internal/protocol/ews/send"
	"github.com/learn0208/mailcli/internal/protocol/ews/show"
)

func searchCmd() *cobra.Command {
	var (
		subject, body, from, to string
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
		Use:   "search",
		Short: "Search messages in a folder (EWS FindItem)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, app, err := loadMergedProfile()
			if err != nil {
				return err
			}
			if err := requireSupportedProtocol(p); err != nil {
				return err
			}
			if err := pews.ValidateProfile(p); err != nil {
				return err
			}
			pw, err := resolvePassword(p)
			if err != nil {
				return err
			}
			c, err := pews.NewHTTPClient(p, resolveUserAgent(app), pw, verbose)
			if err != nil {
				return err
			}
			return search.Run(c, search.Options{
				Subject:             subject,
				Body:                body,
				From:                from,
				To:                  to,
				Since:               since,
				Until:               until,
				Folder:              folder,
				Unread:              unread,
				Read:                read,
				Attachment:          attachments,
				Limit:               limit,
				Output:              output,
				Verbose:             verbose,
				UserSetSince:        cmd.Flags().Changed("since"),
				UserSetUntil:        cmd.Flags().Changed("until"),
				NoDefaultDateWindow: noDefaultDateWindow,
				DefaultDays:         defaultDays,
			})
		},
	}
	cmd.Flags().StringVar(&subject, "subject", "", "subject contains (case-insensitive)")
	cmd.Flags().StringVar(&body, "body", "", "body text contains")
	cmd.Flags().StringVar(&from, "from", "", "from address contains")
	cmd.Flags().StringVar(&to, "to", "", "to recipients contains")
	cmd.Flags().StringVar(&since, "since", "", "received on/after (RFC3339, date, or e.g. \"7 days ago\"); omit with default window: last N days")
	cmd.Flags().StringVar(&until, "until", "", "received on/before")
	cmd.Flags().StringVar(&folder, "folder", "Inbox", "folder name or distinguished id (default Inbox)")
	cmd.Flags().BoolVar(&unread, "unread", false, "only unread")
	cmd.Flags().BoolVar(&read, "read", false, "only read")
	cmd.Flags().BoolVar(&attachments, "attachments", false, "only messages with attachments")
	cmd.Flags().IntVar(&limit, "limit", 10, "max messages (newest first within the query window)")
	cmd.Flags().IntVar(&defaultDays, "default-days", 7, "when neither --since nor --until is given, restrict to the last N calendar days (unless --no-default-date-window)")
	cmd.Flags().BoolVar(&noDefaultDateWindow, "no-default-date-window", false, "search all time when --since/--until are omitted (does not apply if you pass either flag)")
	cmd.Flags().StringVar(&output, "output", "table", "output: table or json")
	return cmd
}

func showCmd() *cobra.Command {
	var itemID, changeKey, format string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show one message body and headers (EWS GetItem)",
		Long:  "Load a message by ItemId from a previous search. Use --format json for machine-readable output.",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, app, err := loadMergedProfile()
			if err != nil {
				return err
			}
			if err := requireSupportedProtocol(p); err != nil {
				return err
			}
			if err := pews.ValidateProfile(p); err != nil {
				return err
			}
			pw, err := resolvePassword(p)
			if err != nil {
				return err
			}
			c, err := pews.NewHTTPClient(p, resolveUserAgent(app), pw, verbose)
			if err != nil {
				return err
			}
			return show.Run(c, show.Options{
				ItemID:    itemID,
				ChangeKey: changeKey,
				Format:    format,
			})
		},
	}
	cmd.Flags().StringVar(&itemID, "item-id", "", "EWS ItemId (from search output)")
	cmd.Flags().StringVar(&changeKey, "change-key", "", "optional ChangeKey from search")
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
		Short: "Send an email message (EWS CreateItem)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, app, err := loadMergedProfile()
			if err != nil {
				return err
			}
			if err := requireSupportedProtocol(p); err != nil {
				return err
			}
			if err := pews.ValidateProfile(p); err != nil {
				return err
			}
			pw, err := resolvePassword(p)
			if err != nil {
				return err
			}
			c, err := pews.NewHTTPClient(p, resolveUserAgent(app), pw, verbose)
			if err != nil {
				return err
			}
			imp, err := send.NormalizeImportance(importance)
			if err != nil {
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
			toAddrs := send.SplitAddresses(toStr)
			res, err := send.Run(c, send.Options{
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
			})
			if err != nil {
				return err
			}
			if !noVerifySent {
				wait := verifySentWaitSec
				if wait < 0 {
					wait = 0
				}
				if wait > 120 {
					wait = 120
				}
				if wait > 0 {
					time.Sleep(time.Duration(wait) * time.Second)
				}
				ref := time.Now()
				hit, verr := search.VerifySentCopy(c, ref, subject, toAddrs, 30*time.Minute, verbose)
				if verr != nil {
					res.VerifyErr = verr.Error()
				} else {
					res.SentVerified = hit.Found
					res.SentVerifyHint = hit.Hint
					if hit.Found {
						res.SentVerifyItemID = hit.Row.ItemID
						res.SentVerifyReceived = hit.Row.DateTimeReceived
					}
				}
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			fmt.Fprintln(os.Stdout, "?????")
			if res.ItemID != "" {
				fmt.Fprintf(os.Stdout, "EWS ?? item_id=%s change_key=%s\n", res.ItemID, res.ChangeKey)
			}
			if !noVerifySent {
				if res.VerifyErr != "" {
					fmt.Fprintf(os.Stderr, "????????%s\n", res.VerifyErr)
					if res.ItemID == "" && res.Note != "" {
						fmt.Fprintln(os.Stdout, res.Note)
					}
				} else if res.SentVerified {
					fmt.Fprintf(os.Stdout, "????????????? %s?item_id=%s\n",
						search.FormatReceivedShort(res.SentVerifyReceived), res.SentVerifyItemID)
				} else if res.SentVerifyHint != "" {
					fmt.Fprintln(os.Stdout, res.SentVerifyHint)
				}
			} else if res.ItemID == "" && res.Note != "" {
				fmt.Fprintln(os.Stdout, res.Note)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&toStr, "to", "", "recipients (comma-separated)")
	cmd.Flags().StringVar(&ccStr, "cc", "", "cc recipients")
	cmd.Flags().StringVar(&bccStr, "bcc", "", "bcc recipients")
	cmd.Flags().StringVar(&subject, "subject", "", "subject line")
	cmd.Flags().StringVar(&textBody, "text", "", "plain text body")
	cmd.Flags().StringVar(&htmlBody, "html", "", "HTML body")
	cmd.Flags().StringSliceVar(&attach, "attach", nil, "attachment file path (repeatable)")
	cmd.Flags().StringVar(&importance, "importance", "", "high | normal | low")
	cmd.Flags().StringVar(&fromAddr, "from", "", "sender address (default: authenticated user)")
	cmd.Flags().StringVar(&fromName, "from-name", "", "UTF-8 sender display name (default: profile display_name; reduces From-name mojibake in some clients)")
	cmd.Flags().BoolVar(&fromNamePlainUTF8, "from-name-plain-utf8", false, "put raw UTF-8 in EWS Name (default: RFC2047 UTF-8 for non-ASCII; use if your server double-encodes)")
	cmd.Flags().BoolVar(&fromAddressOnly, "from-address-only", false, "ASCII From: set sender Name=Email (SMTP); needs full user@domain (--from or smtp_address / inferred user@domain)")
	cmd.Flags().BoolVar(&noVerifySent, "no-verify-sent", false, "do not search Sent Items after send (no wait; for scripts)")
	cmd.Flags().IntVar(&verifySentWaitSec, "verify-sent-wait", 5, "seconds to wait before checking Sent Items (0=no wait; max 120; default 5)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print result as JSON (includes sent verification when not --no-verify-sent)")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("subject")
	return cmd
}
