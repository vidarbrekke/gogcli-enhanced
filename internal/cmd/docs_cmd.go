package cmd

import (
	"github.com/steipete/gogcli/internal/googleapi"
)

var newDocsService = googleapi.NewDocs

// DocsCmd defines the Google Docs command group.
type DocsCmd struct {
	Export DocsExportCmd `cmd:"" name:"export" help:"Export a Google Doc (pdf|docx|txt)"`
	Info   DocsInfoCmd   `cmd:"" name:"info" help:"Get Google Doc metadata"`
	Create DocsCreateCmd `cmd:"" name:"create" help:"Create a Google Doc"`
	Copy   DocsCopyCmd   `cmd:"" name:"copy" help:"Copy a Google Doc"`
	Cat    DocsCatCmd    `cmd:"" name:"cat" help:"Print a Google Doc as plain text"`
	Edit   DocsEditCmd   `cmd:"" name:"edit" help:"Edit Google Doc content"`
}

// DocsEditCmd defines the docs edit subcommands.
type DocsEditCmd struct {
	Append  DocsAppendCmd  `cmd:"" name:"append" help:"Append text to the end of a Google Doc"`
	Batch   DocsBatchCmd   `cmd:"" name:"batch" help:"Apply multiple Docs API edit operations from JSON"`
	Delete  DocsDeleteCmd  `cmd:"" name:"delete" help:"Delete a text range in a Google Doc"`
	Insert  DocsInsertCmd  `cmd:"" name:"insert" help:"Insert text at a specific index in a Google Doc"`
	Replace DocsReplaceCmd `cmd:"" name:"replace" help:"Replace text throughout a Google Doc"`
}

// DocsEditSafetyFlags are shared flags for agentic edit workflows.
type DocsEditSafetyFlags struct {
	DryRun            bool   `name:"dry-run" help:"Build request and print it without executing API call"`
	RequireRevision   string `name:"require-revision" help:"Require this document revision ID for update (optimistic concurrency guard)"`
	OutputRequestFile string `name:"output-request-file" help:"Write normalized request JSON to this file (use '-' for stdout)"`
	Pretty            bool   `name:"pretty" help:"Include normalized pretty-printed request JSON in output"`
}
