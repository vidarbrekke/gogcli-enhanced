package cmd

import (
	"github.com/steipete/gogcli/internal/googleapi"
)

var newDocsService = googleapi.NewDocs

// DocsCmd defines the Google Docs command group.
type DocsCmd struct {
	Export      DocsExportCmd            `cmd:"" name:"export" aliases:"download,dl" help:"Export a Google Doc (pdf|docx|txt)"`
	Info        DocsInfoCmd              `cmd:"" name:"info" aliases:"get,show" help:"Get Google Doc metadata"`
	Create      DocsCreateCmd            `cmd:"" name:"create" aliases:"add,new" help:"Create a Google Doc"`
	Copy        DocsCopyCmd              `cmd:"" name:"copy" aliases:"cp,duplicate" help:"Copy a Google Doc"`
	Cat         DocsCatCmd               `cmd:"" name:"cat" aliases:"text,read" help:"Print a Google Doc as plain text"`
	Comments    DocsCommentsCmd          `cmd:"" name:"comments" help:"Manage comments on a Google Doc"`
	ListTabs    DocsListTabsCmd          `cmd:"" name:"list-tabs" help:"List all tabs in a Google Doc"`
	Positions   DocsPositionsCmd         `cmd:"" name:"positions" help:"Return position helpers (end index, search ranges, headings)"`
	Write       DocsWriteCmd             `cmd:"" name:"write" help:"Write content to a Google Doc"`
	Insert      DocsInlineInsertCmd      `cmd:"" name:"insert" help:"Insert text at a specific position"`
	Delete      DocsInlineDeleteCmd      `cmd:"" name:"delete" help:"Delete text range from document"`
	FindReplace DocsInlineFindReplaceCmd `cmd:"" name:"find-replace" help:"Find and replace text in document"`
	Sed         DocsSedCmd               `cmd:"" name:"sed" help:"Sed-like find-and-replace on Google Docs"`
	Update      DocsUpdateCmd            `cmd:"" name:"update" help:"Update content in a Google Doc"`
	Edit        DocsEditCmd              `cmd:"" name:"edit" help:"Edit Google Doc content"`
}

// DocsEditCmd defines the docs edit subcommands.
type DocsEditCmd struct {
	Append       DocsAppendCmd        `cmd:"" name:"append" help:"Append text to the end of a Google Doc"`
	Batch        DocsBatchCmd         `cmd:"" name:"batch" help:"Apply multiple Docs API edit operations from JSON"`
	Delete       DocsDeleteCmd        `cmd:"" name:"delete" help:"Delete a text range in a Google Doc"`
	Insert       DocsInsertCmd        `cmd:"" name:"insert" help:"Insert text at a specific index in a Google Doc"`
	InsertTable  DocsInsertTableCmd   `cmd:"" name:"insert-table" help:"Insert a table at a specific location in a Google Doc"`
	InsertImage  DocsInsertImageCmd   `cmd:"" name:"insert-image" help:"Insert an image at a specific index in a Google Doc"`
	Locator      DocsLocatorEditCmd   `cmd:"" name:"locator" help:"Apply anchor/marker-based edits in a Google Doc"`
	MergeData    DocsEditMergeDataCmd `cmd:"" name:"merge-data" help:"Generate docs from template using JSON data (mail-merge)"`
	Replace      DocsReplaceCmd       `cmd:"" name:"replace" help:"Replace text throughout a Google Doc"`
	ReplaceImage DocsReplaceImageCmd  `cmd:"" name:"replace-image" help:"Replace an image in a Google Doc with a new image"`
}

// DocsEditSafetyFlags is the shared agentic safety flags for Docs edit commands.
type DocsEditSafetyFlags = AgenticEditSafetyFlags
