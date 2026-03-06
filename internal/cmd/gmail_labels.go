package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"google.golang.org/api/gmail/v1"

	"github.com/steipete/gogcli/internal/backend/gws"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/parity/classify"
	"github.com/steipete/gogcli/internal/parity/normalize"
	"github.com/steipete/gogcli/internal/ui"
)

type GmailLabelsCmd struct {
	List   GmailLabelsListCmd   `cmd:"" name:"list" aliases:"ls" help:"List labels"`
	Get    GmailLabelsGetCmd    `cmd:"" name:"get" aliases:"info,show" help:"Get label details (including counts)"`
	Create GmailLabelsCreateCmd `cmd:"" name:"create" aliases:"add,new" help:"Create a new label"`
	Modify GmailLabelsModifyCmd `cmd:"" name:"modify" aliases:"update,edit,set" help:"Modify labels on threads"`
	Delete GmailLabelsDeleteCmd `cmd:"" name:"delete" aliases:"rm,del" help:"Delete a label"`
}

type GmailLabelsGetCmd struct {
	Label string `arg:"" name:"labelIdOrName" help:"Label ID or name"`
}

func (c *GmailLabelsGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	if Backend() == BackendGWS {
		if err := validateGWSExplicitAccountSelection(flags); err != nil {
			return err
		}
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	if Backend() == BackendGWS {
		return c.runGWSGet(ctx, account)
	}

	svc, err := newGmailService(ctx, account)
	if err != nil {
		return err
	}

	idMap, err := fetchLabelNameToID(svc)
	if err != nil {
		return err
	}
	raw := strings.TrimSpace(c.Label)
	if raw == "" {
		return usage("empty label")
	}
	id := raw
	if v, ok := idMap[strings.ToLower(raw)]; ok {
		id = v
	}

	l, err := svc.Users.Labels.Get("me", id).Context(ctx).Do()
	if err != nil {
		return err
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{"label": l})
	}
	return writeLabelGetText(ctx, labelGetTextView{
		ID:             l.Id,
		Name:           l.Name,
		Type:           l.Type,
		MessagesTotal:  l.MessagesTotal,
		MessagesUnread: l.MessagesUnread,
		ThreadsTotal:   l.ThreadsTotal,
		ThreadsUnread:  l.ThreadsUnread,
	})
}

// gwsFixture adapts gws.Result to classify.FixtureData.
type gwsFixture struct {
	stdout   []byte
	stderr   []byte
	exitCode int
}

func (f gwsFixture) GetStdout() []byte { return f.stdout }
func (f gwsFixture) GetStderr() []byte { return f.stderr }
func (f gwsFixture) GetExitCode() int  { return f.exitCode }

func (c *GmailLabelsGetCmd) runGWSGet(ctx context.Context, _ string) error {
	raw := strings.TrimSpace(c.Label)
	if raw == "" {
		return usage("empty label")
	}
	id := raw
	listRes, err := gws.RunLabelsList(ctx)
	if err != nil {
		return err
	}
	if classify.Classify(gwsFixture{listRes.Stdout, listRes.Stderr, listRes.ExitCode}) == classify.OutcomeOK {
		idMap := gwsLabelsNameToID(listRes.Stdout)
		if v, ok := idMap[strings.ToLower(raw)]; ok {
			id = v
		}
	}

	res, err := gws.RunLabelsGet(ctx, id)
	if err != nil {
		return err
	}
	if classify.Classify(gwsFixture{res.Stdout, res.Stderr, res.ExitCode}) == classify.OutcomeERROR {
		ctxNorm := normalize.InvocationCtx{Service: "gmail", Operation: "labels get", ResourceID: id}
		env, ok := normalize.NormalizeError(res.Stdout, res.Stderr, ctxNorm)
		if !ok {
			return fmt.Errorf("gws labels get failed (exit %d)", res.ExitCode)
		}
		return &BackendError{Env: env}
	}
	if outfmt.IsJSON(ctx) {
		return writeGWSJSON(ctx, res.Stdout, "label")
	}
	return writeGWSLabelsGetTable(ctx, res.Stdout)
}

func gwsLabelsNameToID(stdout []byte) map[string]string {
	var out struct {
		Labels []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"labels"`
	}
	if json.Unmarshal(stdout, &out) != nil {
		return nil
	}
	m := make(map[string]string, len(out.Labels)*2)
	for _, l := range out.Labels {
		if l.ID != "" {
			m[strings.ToLower(l.ID)] = l.ID
			if l.Name != "" {
				m[strings.ToLower(l.Name)] = l.ID
			}
		}
	}
	return m
}

// writeGWSJSON writes gws stdout as JSON. If key is "label" and stdout has no "label" key, wraps top-level as {"label": {...}}.
func writeGWSJSON(ctx context.Context, stdout []byte, resultKey string) error {
	var m map[string]any
	if json.Unmarshal(stdout, &m) != nil {
		return fmt.Errorf("gws output is not valid JSON")
	}
	if resultKey == "label" && m["label"] == nil && (m["id"] != nil || m["name"] != nil) {
		m = map[string]any{"label": m}
	}
	return outfmt.WriteJSON(ctx, stdoutWriter(ctx), m)
}

// writeGWSLabelsGetTable parses gws label JSON and prints ID/NAME/TYPE table.
func writeGWSLabelsGetTable(ctx context.Context, stdout []byte) error {
	var m map[string]any
	if json.Unmarshal(stdout, &m) != nil {
		return fmt.Errorf("gws output is not valid JSON")
	}
	label, _ := m["label"].(map[string]any)
	if label == nil {
		label = m
	}
	return writeLabelGetText(ctx, labelGetTextView{
		ID:             stringValue(label["id"]),
		Name:           stringValue(label["name"]),
		Type:           stringValue(label["type"]),
		MessagesTotal:  int64Value(label["messagesTotal"]),
		MessagesUnread: int64Value(label["messagesUnread"]),
		ThreadsTotal:   int64Value(label["threadsTotal"]),
		ThreadsUnread:  int64Value(label["threadsUnread"]),
	})
}

type labelGetTextView struct {
	ID             string
	Name           string
	Type           string
	MessagesTotal  int64
	MessagesUnread int64
	ThreadsTotal   int64
	ThreadsUnread  int64
}

func writeLabelGetText(ctx context.Context, label labelGetTextView) error {
	w := stdoutWriter(ctx)
	fmt.Fprintf(w, "id\t%s\n", label.ID)
	fmt.Fprintf(w, "name\t%s\n", label.Name)
	fmt.Fprintf(w, "type\t%s\n", label.Type)
	fmt.Fprintf(w, "messages_total\t%d\n", label.MessagesTotal)
	fmt.Fprintf(w, "messages_unread\t%d\n", label.MessagesUnread)
	fmt.Fprintf(w, "threads_total\t%d\n", label.ThreadsTotal)
	fmt.Fprintf(w, "threads_unread\t%d\n", label.ThreadsUnread)
	return nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func int64Value(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(math.Round(n))
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

type GmailLabelsCreateCmd struct {
	Name string `arg:"" help:"Label name"`
}

func (c *GmailLabelsCreateCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	name := strings.TrimSpace(c.Name)
	if name == "" {
		return usage("label name is required")
	}

	svc, err := newGmailService(ctx, account)
	if err != nil {
		return err
	}

	err = ensureLabelNameAvailable(svc, name)
	if err != nil {
		return err
	}

	label, err := createLabel(ctx, svc, name)
	if err != nil {
		return mapLabelCreateError(err, name)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{"label": label})
	}
	u.Out().Printf("Created label: %s (id: %s)", label.Name, label.Id)
	return nil
}

func createLabel(ctx context.Context, svc *gmail.Service, name string) (*gmail.Label, error) {
	return svc.Users.Labels.Create("me", &gmail.Label{
		Name:                  name,
		LabelListVisibility:   "labelShow",
		MessageListVisibility: "show",
	}).Context(ctx).Do()
}

type GmailLabelsListCmd struct{}

func (c *GmailLabelsListCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	if Backend() == BackendGWS {
		if err := validateGWSExplicitAccountSelection(flags); err != nil {
			return err
		}
	}
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	if Backend() == BackendGWS {
		return c.runGWSList(ctx)
	}

	svc, err := newGmailService(ctx, account)
	if err != nil {
		return err
	}

	resp, err := svc.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return err
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{"labels": resp.Labels})
	}
	if len(resp.Labels) == 0 {
		u.Err().Println("No labels")
		return nil
	}

	w, flush := tableWriter(ctx)
	defer flush()
	fmt.Fprintln(w, "ID\tNAME\tTYPE")
	for _, l := range resp.Labels {
		fmt.Fprintf(w, "%s\t%s\t%s\n", l.Id, l.Name, l.Type)
	}
	return nil
}

func (c *GmailLabelsListCmd) runGWSList(ctx context.Context) error {
	res, err := gws.RunLabelsList(ctx)
	if err != nil {
		return err
	}
	if classify.Classify(gwsFixture{res.Stdout, res.Stderr, res.ExitCode}) == classify.OutcomeERROR {
		ctxNorm := normalize.InvocationCtx{Service: "gmail", Operation: "labels list"}
		env, ok := normalize.NormalizeError(res.Stdout, res.Stderr, ctxNorm)
		if !ok {
			return fmt.Errorf("gws labels list failed (exit %d)", res.ExitCode)
		}
		return &BackendError{Env: env}
	}
	if outfmt.IsJSON(ctx) {
		var payload map[string]any
		if json.Unmarshal(res.Stdout, &payload) != nil {
			return fmt.Errorf("gws output is not valid JSON")
		}
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), payload)
	}
	return writeGWSLabelsListTable(ctx, res.Stdout)
}

func writeGWSLabelsListTable(ctx context.Context, stdout []byte) error {
	var out struct {
		Labels []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"labels"`
	}
	if json.Unmarshal(stdout, &out) != nil {
		return fmt.Errorf("gws output is not valid JSON")
	}
	u := ui.FromContext(ctx)
	if u == nil {
		return nil
	}
	if len(out.Labels) == 0 {
		u.Err().Println("No labels")
		return nil
	}
	w, flush := tableWriter(ctx)
	defer flush()
	fmt.Fprintln(w, "ID\tNAME\tTYPE")
	for _, l := range out.Labels {
		fmt.Fprintf(w, "%s\t%s\t%s\n", l.ID, l.Name, l.Type)
	}
	return nil
}

type GmailLabelsModifyCmd struct {
	ThreadIDs []string `arg:"" name:"threadId" help:"Thread IDs"`
	Add       string   `name:"add" help:"Labels to add (comma-separated, name or ID)"`
	Remove    string   `name:"remove" help:"Labels to remove (comma-separated, name or ID)"`
}

func (c *GmailLabelsModifyCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	threadIDs := c.ThreadIDs
	addLabels := splitCSV(c.Add)
	removeLabels := splitCSV(c.Remove)
	if len(addLabels) == 0 && len(removeLabels) == 0 {
		return usage("must specify --add and/or --remove")
	}

	svc, err := newGmailService(ctx, account)
	if err != nil {
		return err
	}

	idMap, err := fetchLabelNameToID(svc)
	if err != nil {
		return err
	}

	addIDs := resolveLabelIDs(addLabels, idMap)
	removeIDs := resolveLabelIDs(removeLabels, idMap)

	type result struct {
		ThreadID string `json:"threadId"`
		Success  bool   `json:"success"`
		Error    string `json:"error,omitempty"`
	}
	results := make([]result, 0, len(threadIDs))

	for _, tid := range threadIDs {
		_, err := svc.Users.Threads.Modify("me", tid, &gmail.ModifyThreadRequest{
			AddLabelIds:    addIDs,
			RemoveLabelIds: removeIDs,
		}).Context(ctx).Do()
		if err != nil {
			results = append(results, result{ThreadID: tid, Success: false, Error: err.Error()})
			if !outfmt.IsJSON(ctx) {
				u.Err().Errorf("%s: %s", tid, err.Error())
			}
			continue
		}
		results = append(results, result{ThreadID: tid, Success: true})
		if !outfmt.IsJSON(ctx) {
			u.Out().Printf("%s\tok", tid)
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{"results": results})
	}
	return nil
}

func fetchLabelNameToID(svc *gmail.Service) (map[string]string, error) {
	resp, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(resp.Labels))
	for _, l := range resp.Labels {
		if l.Id == "" {
			continue
		}
		m[strings.ToLower(l.Id)] = l.Id
		if l.Name != "" {
			m[strings.ToLower(l.Name)] = l.Id
		}
	}
	return m, nil
}

func fetchLabelNameOnlyToID(svc *gmail.Service) (map[string]string, error) {
	resp, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(resp.Labels))
	for _, l := range resp.Labels {
		if l.Id == "" || l.Name == "" {
			continue
		}
		m[strings.ToLower(l.Name)] = l.Id
	}
	return m, nil
}

type GmailLabelsDeleteCmd struct {
	Label string `arg:"" name:"labelIdOrName" help:"Label ID or name"`
}

func (c *GmailLabelsDeleteCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	svc, err := newGmailService(ctx, account)
	if err != nil {
		return err
	}

	raw := strings.TrimSpace(c.Label)
	if raw == "" {
		return usage("empty label")
	}

	// For destructive operations, try exact ID match first before name lookup.
	label, err := svc.Users.Labels.Get("me", raw).Context(ctx).Do()
	if err != nil {
		if !isNotFoundAPIError(err) {
			return err
		}
		// Exact ID not found; resolve by label name only.
		idMap, mapErr := fetchLabelNameOnlyToID(svc)
		if mapErr != nil {
			return mapErr
		}
		id, ok := idMap[strings.ToLower(raw)]
		if !ok {
			return fmt.Errorf("label not found: %s", raw)
		}
		label, err = svc.Users.Labels.Get("me", id).Context(ctx).Do()
		if err != nil {
			return err
		}
	}

	// System labels cannot be deleted
	if label.Type == "system" {
		return fmt.Errorf("cannot delete system label %q", label.Name)
	}

	if confirmErr := confirmDestructive(ctx, flags, fmt.Sprintf("delete label %q", label.Name)); confirmErr != nil {
		return confirmErr
	}

	if err := svc.Users.Labels.Delete("me", label.Id).Context(ctx).Do(); err != nil {
		return err
	}

	return writeResult(ctx, u,
		kv("deleted", true),
		kv("id", label.Id),
		kv("name", label.Name),
	)
}

func fetchLabelIDToName(svc *gmail.Service) (map[string]string, error) {
	resp, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(resp.Labels))
	for _, l := range resp.Labels {
		if l.Id == "" {
			continue
		}
		if l.Name != "" {
			m[l.Id] = l.Name
		} else {
			m[l.Id] = l.Id
		}
	}
	return m, nil
}
