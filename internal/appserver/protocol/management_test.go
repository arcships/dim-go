package protocol_test

import (
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

func TestDecodePermissionWire(t *testing.T) {
	result, err := protocol.ParsePermissionResult(mustRaw(t, `{
		"version":7,
		"policy":{"preset":"workspace-write","allowedTools":["read"],"disallowedTools":[],"future":1},
		"future":2
	}`))
	if err != nil {
		t.Fatalf("permission result: %v", err)
	}
	if result.Version != 7 || result.Policy.Preset != protocol.PermissionPresetWorkspaceWrite {
		t.Fatalf("unexpected result: %+v", result)
	}
	changed, err := protocol.ParsePermissionChanged(mustRaw(t,
		`{"policy":{"preset":"read-only","allowedTools":[],"disallowedTools":[]},"future":1}`))
	if err != nil || changed.Policy.Preset != protocol.PermissionPresetReadOnly {
		t.Fatalf("permission changed: %+v, %v", changed, err)
	}
	if _, err := protocol.ParsePermissionResult(mustRaw(t,
		`{"version":1,"policy":{"preset":"anything","allowedTools":[],"disallowedTools":[]}}`)); err == nil {
		t.Fatal("want an error for an unknown preset")
	}
	if _, err := protocol.ParsePermissionChanged(mustRaw(t, `{}`)); err == nil {
		t.Fatal("want an error for a missing policy")
	}
}

func TestDecodeSessionRename(t *testing.T) {
	result, err := protocol.ParseSessionRenameResult(mustRaw(t,
		`{"title":"Hello","titleSource":"manual","revision":2,"future":1}`))
	if err != nil || result.Title == nil || *result.Title != "Hello" || result.Revision != 2 {
		t.Fatalf("rename: %+v, %v", result, err)
	}
	cleared, err := protocol.ParseSessionRenameResult(mustRaw(t,
		`{"title":null,"titleSource":null,"revision":3}`))
	if err != nil || cleared.Title != nil {
		t.Fatalf("cleared title: %+v, %v", cleared, err)
	}
	if _, err := protocol.ParseSessionRenameResult(mustRaw(t, `{"title":"x","titleSource":"manual"}`)); err == nil {
		t.Fatal("want an error for a missing revision")
	}
}

func TestDecodeStorageAndProgress(t *testing.T) {
	report, err := protocol.ParseStorageReportResult(mustRaw(t,
		`{"databaseBytes":10,"walBytes":2,"reclaimableBytes":0,"future":1}`))
	if err != nil || report.DatabaseBytes != 10 || report.WALBytes != 2 {
		t.Fatalf("storage: %+v, %v", report, err)
	}
	if _, err := protocol.ParseStorageReportResult(mustRaw(t,
		`{"databaseBytes":-1,"walBytes":0,"reclaimableBytes":0}`)); err == nil {
		t.Fatal("want an error for a negative byte count")
	}
	progress, err := protocol.ParseManagementProgressNotice(mustRaw(t,
		`{"id":7,"method":"plugin.install","data":{"phase":"fetch"},"future":1}`))
	if err != nil || progress.Method != "plugin.install" || string(progress.Data) != `{"phase":"fetch"}` {
		t.Fatalf("progress: %+v, %v", progress, err)
	}
	if _, err := protocol.ParseManagementProgressNotice(mustRaw(t, `{"id":1,"data":{}}`)); err == nil {
		t.Fatal("want an error for a missing method")
	}
}

func TestDecodePluginWire(t *testing.T) {
	list, err := protocol.ParsePluginListResult(mustRaw(t, `[{
		"name":"Attachments","use":"builtin","off":false,"locked":false,
		"candidates":[{"id":"builtin","kinds":["code"],"future":1}],
		"status":"running","future":2
	}]`))
	if err != nil || len(list) != 1 || list[0].Name != "Attachments" {
		t.Fatalf("plugin list: %+v, %v", list, err)
	}
	if _, err := protocol.ParsePluginListResult(mustRaw(t,
		`[{"name":"x","locked":false,"candidates":[]}]`)); err != nil {
		t.Fatalf("off plugin without use: %v", err)
	}
	if _, err := protocol.ParsePluginListResult(mustRaw(t, `[{"candidates":[]}]`)); err == nil {
		t.Fatal("want an error for a missing name")
	}
	if _, err := protocol.ParsePluginListResult(mustRaw(t, `[{"name":"x"}]`)); err == nil {
		t.Fatal("want an error for missing candidates")
	}
	empty, err := protocol.ParsePluginOperationResult(mustRaw(t, `{}`))
	if err != nil || empty.Status != "" {
		t.Fatalf("empty operation: %+v, %v", empty, err)
	}
	running, err := protocol.ParsePluginOperationResult(mustRaw(t, `{"status":"running","future":1}`))
	if err != nil || running.Status != "running" {
		t.Fatalf("operation: %+v, %v", running, err)
	}
}

func TestDecodeSkillsWire(t *testing.T) {
	catalog, err := protocol.ParseSkillsCatalogResult(mustRaw(t, `{
		"version":3,
		"entries":[{
			"id":"user:demo","source":"user","name":"demo","description":"d",
			"filePath":"/tmp/SKILL.md","allowedTools":[],"enabled":true,"shadowedBy":null,"future":1
		}],
		"issues":[],
		"future":2
	}`))
	if err != nil || len(catalog.Entries) != 1 || catalog.Entries[0].FilePath == "" {
		t.Fatalf("catalog: %+v, %v", catalog, err)
	}
	if _, err := protocol.ParseSkillsCatalogResult(mustRaw(t,
		`{"version":1,"entries":[{"id":"x","name":"x"}],"issues":[]}`)); err == nil {
		t.Fatal("want an error for an entry without filePath")
	}
	if _, err := protocol.ParseSkillsRefreshResult(mustRaw(t, `{"version":1}`)); err == nil {
		t.Fatal("want an error for missing issues")
	}
	installations, err := protocol.ParseSkillsInstallationsResult(mustRaw(t, `{
		"installations":[{
			"id":"i","source":{"path":"/tmp/skills"},"revision":"r",
			"skills":[{"path":"a","id":"a"}],"future":1
		}]
	}`))
	if err != nil || installations.Installations[0].ID != "i" {
		t.Fatalf("installations: %+v, %v", installations, err)
	}
	if _, err := protocol.ParseSkillsInstallationsResult(mustRaw(t,
		`{"installations":[{"source":{}}]}`)); err == nil {
		t.Fatal("want an error for an installation without identity")
	}
}

func TestDecodeHooksWire(t *testing.T) {
	catalog, err := protocol.ParseHooksCatalogResult(mustRaw(t, `{
		"userRevision":"r1","projectsRevision":"r2",
		"userDocument":{"hooks":{},"disableAllHooks":false},
		"disableAllHooks":false,
		"sources":[{"source":"user","enabled":true,"state":"ok","future":1}],
		"hooks":[],
		"future":2
	}`))
	if err != nil || catalog.UserRevision != "r1" || len(catalog.Sources) != 1 {
		t.Fatalf("hooks catalog: %+v, %v", catalog, err)
	}
	if _, err := protocol.ParseHooksCatalogResult(mustRaw(t,
		`{"userRevision":"r1","projectsRevision":"r2","sources":[],"hooks":[]}`)); err == nil {
		t.Fatal("want an error for a missing userDocument")
	}
	saved, err := protocol.ParseHooksSaveResult(mustRaw(t, `{"revision":"r3","future":1}`))
	if err != nil || saved.Revision != "r3" {
		t.Fatalf("hooks save: %+v, %v", saved, err)
	}
	if _, err := protocol.ParseHooksSaveResult(mustRaw(t, `{}`)); err == nil {
		t.Fatal("want an error for a missing revision")
	}
}

func TestDecodeMCPWire(t *testing.T) {
	catalog, err := protocol.ParseMCPCatalogResult(mustRaw(t, `{
		"userRevision":"u1","projectRevision":null,
		"userApplication":{"appliedRevision":"u1"},"pluginApplication":{},"projectApplication":null,
		"servers":[{"id":"demo","source":"user","enabled":true,"transport":"stdio","future":1}],
		"invalid":[],
		"future":2
	}`))
	if err != nil || len(catalog.Servers) != 1 || catalog.Servers[0].ID != "demo" {
		t.Fatalf("mcp catalog: %+v, %v", catalog, err)
	}
	if _, err := protocol.ParseMCPCatalogResult(mustRaw(t, `{
		"userRevision":"u1","servers":[{"id":"x","source":"elsewhere","transport":"stdio"}],"invalid":[]
	}`)); err == nil {
		t.Fatal("want an error for an unknown server source")
	}
	sources, err := protocol.ParseMCPImportSourcesResult(mustRaw(t, `{"sources":[
		{"id":"claude-code:user","client":"claude-code","scope":"user","path":"~/.claude.json","state":"ready","entries":1,"future":1}
	]}`))
	if err != nil || sources.Sources[0].Entries != 1 {
		t.Fatalf("import sources: %+v, %v", sources, err)
	}
	if _, err := protocol.ParseMCPImportSourcesResult(mustRaw(t, `{"sources":[{"id":"x"}]}`)); err == nil {
		t.Fatal("want an error for an import source without client")
	}
	config, err := protocol.ParseMCPConfigurationResult(mustRaw(t,
		`{"enabled":true,"secretsInUserScope":true,"configuration":{"command":"x"},"future":1}`))
	if err != nil || string(config.Configuration) != `{"command":"x"}` {
		t.Fatalf("mcp configuration: %+v, %v", config, err)
	}
	if _, err := protocol.ParseMCPConfigurationResult(mustRaw(t, `{"enabled":true}`)); err == nil {
		t.Fatal("want an error for a missing configuration")
	}
}

func TestDecodeScheduleWire(t *testing.T) {
	list, err := protocol.ParseScheduleListResult(mustRaw(t, `{"jobs":[{
		"id":"j1","name":"job","schedule":{"every_minutes":60},"prompt":"p","cwd":"/tmp",
		"mode":"create","policy_cap":{"preset":"workspace-write"},"enabled":false,
		"max_run_minutes":60,"revision":1,"created_by":"user","future":1
	}]}`))
	if err != nil || len(list.Jobs) != 1 || list.Jobs[0].Mode != "create" {
		t.Fatalf("schedule list: %+v, %v", list, err)
	}
	if _, err := protocol.ParseScheduleListResult(mustRaw(t,
		`{"jobs":[{"name":"job","prompt":"p","cwd":"/tmp","mode":"create","policy_cap":{}}]}`)); err == nil {
		t.Fatal("want an error for a job without id")
	}
	saved, err := protocol.ParseScheduleSaveResult(mustRaw(t, `{
		"transition":{"from":null,"to":{"origin":"user"}},
		"job":{"id":"j1","name":"job","schedule":{"every_minutes":60},"prompt":"p","cwd":"/tmp",
			"mode":"create","policy_cap":{},"enabled":false,"max_run_minutes":60,"revision":1,"created_by":"user"}
	}`))
	if err != nil || saved.Job.ID != "j1" {
		t.Fatalf("schedule save: %+v, %v", saved, err)
	}
	history, err := protocol.ParseScheduleHistoryResult(mustRaw(t, `{"runs":[{
		"run_id":"r1","job_id":"j1","window":"2026-01-01T00:00:00.000Z","state":"completed",
		"submission_id":"s1","created_at":"2026-01-01T00:00:00.000Z","future":1
	}]}`))
	if err != nil || history.Runs[0].RunID != "r1" {
		t.Fatalf("schedule history: %+v, %v", history, err)
	}
	if _, err := protocol.ParseScheduleHistoryResult(mustRaw(t,
		`{"runs":[{"job_id":"j1","state":"completed"}]}`)); err == nil {
		t.Fatal("want an error for a run without run_id")
	}
	item, err := protocol.ParseScheduleWatchItem(mustRaw(t, `{"kind":"watching","seq":0,"future":1}`))
	if err != nil || item.Kind != "watching" {
		t.Fatalf("watch item: %+v, %v", item, err)
	}
}

func TestDecodeModalityWire(t *testing.T) {
	capabilities, err := protocol.ParseModalityCapabilitiesResult(mustRaw(t, `{
		"revision":"r1","future":1,
		"categories":[{
			"category":"image","available":false,
			"reasons":[{"dimension":"adapter","detail":"none"}],
			"operations":[{
				"operation":"generate","key":"modality.image.generate","available":false,
				"reasons":[],"default":null,"effective":null,"options":[],"future":2
			}]
		}]
	}`))
	if err != nil || len(capabilities.Categories) != 1 {
		t.Fatalf("capabilities: %+v, %v", capabilities, err)
	}
	if _, err := protocol.ParseModalityCapabilitiesResult(mustRaw(t,
		`{"revision":"r1","categories":[{"category":"image"}]}`)); err == nil {
		t.Fatal("want an error for a category without reasons")
	}
	reply, err := protocol.ParseModalityTaskReply(mustRaw(t, `{"task":{
		"version":1,"task_id":"t1","client_task_id":null,"revision":1,
		"category":"image","operation":"generate","executor":{},"params":{},
		"state":"queued","remote_handle":null,"progress":null,"result":null,"error":null,
		"remote_cancel":null,"created_at":"a","updated_at":"b","finished_at":null,"future":1
	}}`))
	if err != nil || reply.Task.TaskID != "t1" {
		t.Fatalf("task reply: %+v, %v", reply, err)
	}
	if _, err := protocol.ParseModalityTaskReply(mustRaw(t,
		`{"task":{"task_id":"t1","category":"image","operation":"generate","state":"unknown"}}`)); err == nil {
		t.Fatal("want an error for an unknown task state")
	}
	list, err := protocol.ParseModalityTaskListResult(mustRaw(t, `{"tasks":[],"next":"t1"}`))
	if err != nil || list.Next == nil || *list.Next != "t1" {
		t.Fatalf("task list: %+v, %v", list, err)
	}
	page, err := protocol.ParseModalityTaskResultReply(mustRaw(t,
		`{"data_base64":"","eof":true,"future":1}`))
	if err != nil || !page.EOF {
		t.Fatalf("task result: %+v, %v", page, err)
	}
	if _, err := protocol.ParseModalityTaskResultReply(mustRaw(t, `{"data_base64":""}`)); err == nil {
		t.Fatal("want an error for a result page with neither data nor eof")
	}
	opened, err := protocol.ParseModalityRealtimeOpenResult(mustRaw(t,
		`{"connectionId":"c1","transport":"relay","future":1}`))
	if err != nil || opened.ConnectionID != "c1" {
		t.Fatalf("realtime open: %+v, %v", opened, err)
	}
	if _, err := protocol.ParseModalityRealtimeOpenResult(mustRaw(t,
		`{"connectionId":"c1","transport":"webrtc"}`)); err == nil {
		t.Fatal("want an error for an unknown transport")
	}
	event, err := protocol.ParseModalityRealtimeEventNotice(mustRaw(t,
		`{"connectionId":"c1","event":{"type":"text"},"future":1}`))
	if err != nil || string(event.Event) != `{"type":"text"}` {
		t.Fatalf("realtime event: %+v, %v", event, err)
	}
	if _, err := protocol.ParseModalityRealtimeEventNotice(mustRaw(t, `{"connectionId":"c1"}`)); err == nil {
		t.Fatal("want an error for an event notice without event")
	}
	ended, err := protocol.ParseModalityRealtimeEndedNotice(mustRaw(t,
		`{"connectionId":"c1","reason":"closed","future":1}`))
	if err != nil || ended.Reason != "closed" {
		t.Fatalf("realtime ended: %+v, %v", ended, err)
	}
	if _, err := protocol.ParseModalityRealtimeEndedNotice(mustRaw(t, `{"connectionId":"c1"}`)); err == nil {
		t.Fatal("want an error for an ended notice without reason")
	}
}

func TestDecodeShellAndWakeupWire(t *testing.T) {
	run, err := protocol.ParseShellRunResult(mustRaw(t, `{
		"runId":"r1","status":"completed","output":"hi\n","truncated":false,
		"durationMs":5,"exitCode":0,"future":1
	}`))
	if err != nil || run.RunID != "r1" || run.ExitCode == nil || *run.ExitCode != 0 {
		t.Fatalf("shell run: %+v, %v", run, err)
	}
	if _, err := protocol.ParseShellRunResult(mustRaw(t, `{"status":"completed","output":""}`)); err == nil {
		t.Fatal("want an error for a run without runId")
	}
	if _, err := protocol.ParseShellRunResult(mustRaw(t,
		`{"runId":"r1","status":"weird","output":""}`)); err == nil {
		t.Fatal("want an error for an unknown shell status")
	}
	list, err := protocol.ParseShellListResult(mustRaw(t, `{"runs":[
		{"runId":"r1","branchId":"main","mode":"record","command":"ls","startedAt":1,"future":1}
	]}`))
	if err != nil || len(list.Runs) != 1 {
		t.Fatalf("shell list: %+v, %v", list, err)
	}
	if _, err := protocol.ParseShellListResult(mustRaw(t, `{"runs":[{"runId":"r1"}]}`)); err == nil {
		t.Fatal("want an error for a run without command")
	}
	created, err := protocol.ParseSessionWakeupCreateResult(mustRaw(t,
		`{"wakeupId":"w1","future":1}`))
	if err != nil || created.WakeupID != "w1" {
		t.Fatalf("wakeup create: %+v, %v", created, err)
	}
	if _, err := protocol.ParseSessionWakeupCreateResult(mustRaw(t, `{}`)); err == nil {
		t.Fatal("want an error for a missing wakeupId")
	}
	wakeups, err := protocol.ParseSessionWakeupListResult(mustRaw(t, `{"wakeups":[
		{"wakeupId":"w1","at":"2026-01-01T00:00:00.000Z","prompt":"p","state":"scheduled","future":1}
	]}`))
	if err != nil || wakeups.Wakeups[0].State != "scheduled" {
		t.Fatalf("wakeup list: %+v, %v", wakeups, err)
	}
	if _, err := protocol.ParseSessionWakeupListResult(mustRaw(t,
		`{"wakeups":[{"wakeupId":"w1","at":"x","state":"weird"}]}`)); err == nil {
		t.Fatal("want an error for an unknown wakeup state")
	}
	notice := protocol.SessionNotice{
		SessionID: "s", Source: "Scheduler", Kind: "wakeup",
		Data: mustRaw(t, `{"branchId":"main","wakeupId":"w1","state":"fired","future":1}`),
	}
	data, err := notice.ParseWakeup()
	if err != nil || data.WakeupID != "w1" || data.State != "fired" {
		t.Fatalf("wakeup notice: %+v, %v", data, err)
	}
}
