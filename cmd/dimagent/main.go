// Command dimagent is the MyGo native UI prototype entry point.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

func main() {
	// The fixture stands in for the App Server projection: it is view state
	// only, and goes away once Milestone 3 lands.
	state := &dimui.State{
		Sessions: []dimui.Session{
			{ID: "s1", Title: "Fixture session"},
			{ID: "s2", Title: "Second fixture"},
		},
		Selected: "s1",
		// The run and the tool call below are fixture values, not reports: the
		// supervisor and the timeline projection replace them in Milestone 2
		// and 3, and nothing in the shell derives them.
		Run: dimui.RunRunning,
		Messages: []dimui.Message{
			{ID: "m1", Author: dimui.AuthorUser, Content: "dimagent-mygo: native UI prototype"},
			{ID: "m2", Author: dimui.AuthorAssistant, Content: "## Native shell\n\nA keyed `ui.List` of rows, the assistant's in **Markdown**."},
			{ID: "m3", Author: dimui.AuthorTool, Tool: dimui.Tool{
				Name:   "bash",
				Params: `{"command":"go test ./internal/ui -count=1 -timeout 120s"}`,
				Status: dimui.ToolRunning,
			}},
			{ID: "m4", Author: dimui.AuthorApproval, Content: "Run the release script?"},
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "DimAgent",
			Width:   1080,
			Height:  720,
			Content: ui.View(state.View),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
