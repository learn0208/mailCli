package search

import (
	"fmt"
	"os"
	"time"

	"github.com/tschuyebuhl/ews"
)

// FindItems runs FindItem and returns rows without printing (for programmatic use / send verify).
func FindItems(c ews.Client, opts Options) ([]Row, error) {
	normalizeSearchOptions(&opts)
	ref := time.Now()
	applyDefaultDateWindow(&opts, ref)
	body, err := buildFindItemBody(opts, ref)
	if err != nil {
		return nil, err
	}
	raw, err := c.SendAndReceive(body)
	if err != nil && isSubjectOnly(opts) && isInvalidRequest(err) {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, "Contains search rejected, retrying with QueryString...")
		}
		raw, err = c.SendAndReceive([]byte(buildQueryStringFindItem(opts)))
	}
	if err != nil {
		return nil, classifyErr(err)
	}
	rows, err := parseFindItemResponse(raw)
	if err != nil {
		return nil, err
	}
	return enrichRows(c, rows, opts.Verbose)
}
