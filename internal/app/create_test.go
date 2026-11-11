package app

import (
	"context"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

// TestCreateSessionPassesPermissionPreset 核对新建会话显式带权限档位。
func TestCreateSessionPassesPermissionPreset(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.create", Result: map[string]any{"sessionId": testSession}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	if _, err := result.base.CreateSession(ctx, t.TempDir(), "full-access"); err != nil {
		t.Fatalf("新建会话: %v", err)
	}
	params := requestParams(t, server.RequestsFor("session.create")[0])
	policy, _ := params["permissionPolicy"].(map[string]any)
	if policy["preset"] != "full-access" {
		t.Fatalf("新建会话应带权限档位: %v", params)
	}

	// 空档位表示按服务端默认，不猜、不带字段。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.create", Result: map[string]any{"sessionId": testSession}},
		},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	if _, err := result2.base.CreateSession(ctx, t.TempDir(), ""); err != nil {
		t.Fatalf("新建会话: %v", err)
	}
	params2 := requestParams(t, server2.RequestsFor("session.create")[0])
	if _, ok := params2["permissionPolicy"]; ok {
		t.Fatalf("空档位不应带 permissionPolicy: %v", params2)
	}
}
