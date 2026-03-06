package cmd

import (
	"context"
	"io"
	"os"
	"text/tabwriter"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// stdoutWriter returns the stdout writer from the UI in ctx when present, otherwise os.Stdout.
// Use this for all command output so tests that replace os.Stdout or use a UI with io.Discard get correct behavior.
func stdoutWriter(ctx context.Context) io.Writer {
	if u := ui.FromContext(ctx); u != nil {
		return u.Stdout()
	}

	return os.Stdout
}

type resultKV struct {
	Key   string
	Value any
}

func kv(key string, value any) resultKV {
	return resultKV{Key: key, Value: value}
}

func tableWriter(ctx context.Context) (io.Writer, func()) {
	if outfmt.IsPlain(ctx) {
		return stdoutWriter(ctx), func() {}
	}
	tw := tabwriter.NewWriter(stdoutWriter(ctx), 0, 4, 2, ' ', 0)
	return tw, func() { _ = tw.Flush() }
}

func writeResult(ctx context.Context, u *ui.UI, kvs ...resultKV) error {
	if outfmt.IsJSON(ctx) {
		m := make(map[string]any, len(kvs))
		for _, kv := range kvs {
			m[kv.Key] = kv.Value
		}
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), m)
	}
	if u == nil {
		return nil
	}
	for _, kv := range kvs {
		switch v := kv.Value.(type) {
		case bool:
			u.Out().Printf("%s\t%t", kv.Key, v)
		default:
			u.Out().Printf("%s\t%v", kv.Key, kv.Value)
		}
	}
	return nil
}

func printNextPageHint(u *ui.UI, nextPageToken string) {
	if u == nil || nextPageToken == "" {
		return
	}
	u.Err().Printf("# Next page: --page %s", nextPageToken)
}
