package app

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/learn0208/mailcli/internal/config"
	"github.com/learn0208/mailcli/internal/domain"
	pews "github.com/learn0208/mailcli/internal/protocol/ews"
	"github.com/learn0208/mailcli/internal/protocol/ews/search"
	"github.com/learn0208/mailcli/internal/protocol/ews/send"
	"github.com/learn0208/mailcli/internal/protocol/ews/show"
	imapproto "github.com/learn0208/mailcli/internal/protocol/imap"
	imapsearch "github.com/learn0208/mailcli/internal/protocol/imap/search"
	imapshow "github.com/learn0208/mailcli/internal/protocol/imap/show"
	smtpproto "github.com/learn0208/mailcli/internal/protocol/smtp"
	smtpsend "github.com/learn0208/mailcli/internal/protocol/smtp/send"
)

func requireSupportedProtocol(p config.Profile) error {
	proto := domain.NormalizeProtocol(p.Protocol)
	if !proto.Supported() {
		return fmt.Errorf("protocol %q is not supported (available: ews, imap)", p.Protocol)
	}
	return nil
}

func runEWSSearch(p config.Profile, app config.AppSettings, opts search.Options) error {
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
	return search.Run(c, opts)
}

func runIMAPSearch(p config.Profile, opts imapsearch.Options) error {
	if err := imapproto.ValidateProfile(p); err != nil {
		return err
	}
	pw, err := resolvePassword(p)
	if err != nil {
		return err
	}
	c, err := imapproto.Connect(p, pw)
	if err != nil {
		return err
	}
	defer c.Close()
	return imapsearch.Run(c, opts)
}

func runEWSShow(p config.Profile, app config.AppSettings, opts show.Options) error {
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
	return show.Run(c, opts)
}

func runIMAPShow(p config.Profile, folder string, opts imapshow.Options) error {
	if err := imapproto.ValidateProfile(p); err != nil {
		return err
	}
	pw, err := resolvePassword(p)
	if err != nil {
		return err
	}
	c, err := imapproto.Connect(p, pw)
	if err != nil {
		return err
	}
	defer c.Close()
	opts.Folder = folder
	return imapshow.Run(c, opts)
}

func runEWSSend(p config.Profile, app config.AppSettings, opts send.Options, noVerify bool, verifyWaitSec int, jsonOut bool) error {
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
	res, err := send.Run(c, opts)
	if err != nil {
		return err
	}
	if !noVerify {
		wait := verifyWaitSec
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
		hit, verr := search.VerifySentCopy(c, ref, opts.Subject, opts.To, 30*time.Minute, verbose)
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
	return printSendResult(res, jsonOut, noVerify, true)
}

func runSMTPSend(p config.Profile, opts smtpsend.Options, noVerify bool, verifyWaitSec int, jsonOut bool) error {
	if err := smtpproto.ValidateProfile(p); err != nil {
		return err
	}
	pw, err := resolvePassword(p)
	if err != nil {
		return err
	}
	res, err := smtpsend.Run(p, pw, opts)
	if err != nil {
		return err
	}
	if !noVerify {
		if err := imapproto.ValidateProfile(p); err != nil {
			return fmt.Errorf("imap settings required for sent verification: %w", err)
		}
		wait := verifyWaitSec
		if wait < 0 {
			wait = 0
		}
		if wait > 120 {
			wait = 120
		}
		vres, _ := smtpsend.VerifySent(p, pw, opts.Subject, opts.To, time.Duration(wait)*time.Second)
		if vres != nil {
			res.SentVerified = vres.SentVerified
			res.SentVerifyItemID = vres.SentVerifyItemID
			res.SentVerifyReceived = vres.SentVerifyReceived
			res.SentVerifyHint = vres.SentVerifyHint
			res.VerifyErr = vres.VerifyErr
		}
	}
	return printSMTPResult(res, jsonOut, noVerify)
}

func printSendResult(res *send.Result, jsonOut, noVerify, ews bool) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	fmt.Fprintln(os.Stdout, "发送成功")
	if ews && res.ItemID != "" {
		fmt.Fprintf(os.Stdout, "EWS 项 item_id=%s change_key=%s\n", res.ItemID, res.ChangeKey)
	}
	if !noVerify {
		if res.VerifyErr != "" {
			fmt.Fprintf(os.Stderr, "已发送文件夹校验失败：%s\n", res.VerifyErr)
		} else if res.SentVerified {
			fmt.Fprintf(os.Stdout, "已在已发送文件夹确认副本（%s）uid=%s\n",
				search.FormatReceivedShort(res.SentVerifyReceived), res.SentVerifyItemID)
		} else if res.SentVerifyHint != "" {
			fmt.Fprintln(os.Stdout, res.SentVerifyHint)
		}
	} else if res.Note != "" {
		fmt.Fprintln(os.Stdout, res.Note)
	}
	return nil
}

func printSMTPResult(res *smtpsend.Result, jsonOut, noVerify bool) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	fmt.Fprintln(os.Stdout, "发送成功")
	if res.Note != "" {
		fmt.Fprintln(os.Stdout, res.Note)
	}
	if !noVerify {
		if res.VerifyErr != "" {
			fmt.Fprintf(os.Stderr, "已发送文件夹校验失败：%s\n", res.VerifyErr)
		} else if res.SentVerified {
			fmt.Fprintf(os.Stdout, "已在已发送文件夹确认副本 uid=%s\n", res.SentVerifyItemID)
		} else if res.SentVerifyHint != "" {
			fmt.Fprintln(os.Stdout, res.SentVerifyHint)
		}
	}
	return nil
}

func splitAddresses(s string) []string {
	return smtpsend.SplitAddresses(s)
}
