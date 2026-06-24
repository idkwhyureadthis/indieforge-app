package antivirus

import (
	"context"
	"io"

	clamd "github.com/dutchcoders/go-clamd"
)

// ClamAV scans streams against a clamd instance.
type ClamAV struct{ addr string }

// NewClamAV builds a scanner backed by a clamd daemon at addr (e.g. "tcp://clamav:3310").
func NewClamAV(addr string) *ClamAV { return &ClamAV{addr: addr} }

// Scan reports whether the stream is clean; on a hit it returns the signature.
func (c *ClamAV) Scan(_ context.Context, r io.Reader) (clean bool, signature string, err error) {
	client := clamd.NewClamd(c.addr)
	abort := make(chan bool)
	defer close(abort)
	results, err := client.ScanStream(r, abort)
	if err != nil {
		return false, "", err
	}
	for res := range results {
		if res.Status == clamd.RES_FOUND {
			return false, res.Description, nil
		}
		if res.Status == clamd.RES_ERROR {
			return false, "", io.ErrUnexpectedEOF
		}
	}
	return true, "", nil
}

// Noop is used when antivirus is disabled (no CLAMAV_ADDR configured).
type Noop struct{}

// NewNoop builds a Scanner that always reports clean, for when CLAMAV_ADDR is unset.
func NewNoop() *Noop { return &Noop{} }

// Scan drains r and always reports the file as clean.
func (Noop) Scan(_ context.Context, r io.Reader) (bool, string, error) {
	_, _ = io.Copy(io.Discard, r)
	return true, "", nil
}
