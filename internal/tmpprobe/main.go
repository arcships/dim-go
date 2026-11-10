package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
)

func keys(raw json.RawMessage) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		var a []json.RawMessage
		if err := json.Unmarshal(raw, &a); err == nil {
			if len(a) == 0 {
				return "[]"
			}
			return "array[" + itoa(len(a)) + "] item:" + keys(a[0])
		}
		return "?"
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return fmt.Sprint(out)
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

func main() {
	dataDir, _ := os.MkdirTemp("/tmp", "d12p-")
	defer os.RemoveAll(dataDir)
	// 复制夹具 home（来源只读）：config.json + rust/ 供服务端启动。
	if err := copyTree("/private/tmp/dimagent-real-model/home", dataDir); err != nil {
		fmt.Println("copy fixture:", err)
		return
	}
	sup, err := daemon.New(daemon.Config{
		DataDir:    dataDir,
		ServerPath: os.Getenv("HOME") + "/.dimcode-alpha/rust/dimcode-server",
		ServerArgs: []string{"app-server"},
		MockModel:  false,
		ClientName: "d12-probe",
	})
	if err != nil {
		panic(err)
	}
	defer sup.Close(nil)
	conn, err := sup.Connect(context.Background())
	if err != nil {
		panic(err)
	}
	defer sup.Shutdown(context.Background())
	client := conn.Client
	ctx := context.Background()
	var providerFeatures []string
	for _, feature := range conn.Hello.Features {
		if strings.HasPrefix(feature, "provider") || strings.HasPrefix(feature, "collaboration") || strings.HasPrefix(feature, "agent") || strings.HasPrefix(feature, "session.agent") {
			providerFeatures = append(providerFeatures, feature)
		}
	}
	fmt.Println("hello features (provider/collaboration/agent):", providerFeatures)

	list, err := client.Call(ctx, "provider.list", nil)
	if err != nil {
		fmt.Println("provider.list err:", err)
		return
	}
	fmt.Println("provider.list:", keys(list))

	var parsed struct {
		Providers []map[string]json.RawMessage `json:"providers"`
	}
	_ = json.Unmarshal(list, &parsed)
	fmt.Println("provider.list providers:", len(parsed.Providers))
	if len(parsed.Providers) > 0 {
		first := parsed.Providers[0]
		ids := make([]string, 0, len(first))
		for k := range first {
			ids = append(ids, k)
		}
		sort.Strings(ids)
		fmt.Println("provider item keys:", ids)
		var id struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(mustMarshal(first), &id)
		if id.ID == "" {
			var alt struct {
				ProviderID string `json:"providerId"`
			}
			_ = json.Unmarshal(mustMarshal(first), &alt)
			id.ID = alt.ProviderID
		}
		fmt.Println("provider id empty:", id.ID == "")
		if id.ID != "" {
			mgmt, err := client.Call(ctx, "provider.management", map[string]any{"providerId": id.ID})
			if err != nil {
				fmt.Println("provider.management err:", err)
			} else {
				fmt.Println("provider.management:", keys(mgmt))
				var detail struct {
					Providers []map[string]json.RawMessage `json:"providers"`
				}
				_ = json.Unmarshal(mgmt, &detail)
				if len(detail.Providers) > 0 {
					entry := detail.Providers[0]
					names := make([]string, 0, len(entry))
					for k := range entry {
						names = append(names, k)
					}
					sort.Strings(names)
					fmt.Println("provider.management item keys:", names)
					for _, sub := range []string{"connection", "account", "preferences"} {
						if raw, ok := entry[sub]; ok {
							fmt.Println("  ", sub, "keys:", keys(raw))
						}
					}
					if raw, ok := entry["models"]; ok {
						var models []map[string]json.RawMessage
						_ = json.Unmarshal(raw, &models)
						if len(models) > 0 {
							names := make([]string, 0, len(models[0]))
							for k := range models[0] {
								names = append(names, k)
							}
							sort.Strings(names)
							fmt.Println("   model item keys:", names)
						}
					}
				}
			}
		}
	}

	policy, err := client.Call(ctx, "collaboration.model_policy", nil)
	if err != nil {
		fmt.Println("collaboration.model_policy err:", err)
	} else {
		fmt.Println("collaboration.model_policy:", keys(policy))
		var parsed struct {
			Policy map[string]json.RawMessage `json:"policy"`
		}
		_ = json.Unmarshal(policy, &parsed)
		fmt.Println("collaboration policy keys:", keys(mustMarshal(parsed.Policy)))
	}
	agents, err := client.Call(ctx, "agent.list", nil)
	if err != nil {
		fmt.Println("agent.list err:", err)
	} else {
		fmt.Println("agent.list:", keys(agents))
	}
}

func mustMarshal(m map[string]json.RawMessage) json.RawMessage {
	raw, _ := json.Marshal(m)
	return raw
}

// copyTree 递归复制目录（处理 /tmp 夹具的浅层结构）。
func copyTree(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from := src + "/" + entry.Name()
		to := dst + "/" + entry.Name()
		if entry.IsDir() {
			if err := os.MkdirAll(to, 0o700); err != nil {
				return err
			}
			if err := copyTree(from, to); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			return err
		}
		if err := os.WriteFile(to, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}
