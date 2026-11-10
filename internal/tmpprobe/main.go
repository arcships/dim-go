package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
)

func main() {
	dataDir, _ := os.MkdirTemp("/tmp", "d12p-")
	defer os.RemoveAll(dataDir)
	if err := copyTree("/private/tmp/dimagent-real-model/home", dataDir); err != nil {
		fmt.Println("copy:", err)
		return
	}
	sup, err := daemon.New(daemon.Config{
		DataDir:    dataDir,
		ServerPath: os.Getenv("HOME") + "/.dimcode-alpha/rust/dimcode-server",
		ServerArgs: []string{"app-server"},
		ClientName: "d12-probe",
	})
	if err != nil {
		fmt.Println("new:", err)
		return
	}
	defer sup.Close(nil)
	conn, err := sup.Connect(context.Background())
	if err != nil {
		fmt.Println("connect:", err)
		return
	}
	defer sup.Shutdown(context.Background())
	ctx := context.Background()
	raw, err := conn.Client.Call(ctx, "provider.list", map[string]any{})
	if err != nil {
		fmt.Println("list:", err)
		return
	}
	var list struct {
		Providers []struct {
			ProviderID string `json:"providerId"`
		} `json:"providers"`
	}
	_ = json.Unmarshal(raw, &list)
	for index, entry := range list.Providers {
		if index >= 5 {
			break
		}
		mgmt, err := conn.Client.Call(ctx, "provider.management", map[string]any{"providerId": entry.ProviderID})
		if err != nil {
			fmt.Println("mgmt:", err)
			continue
		}
		var parsed struct {
			Providers []map[string]json.RawMessage `json:"providers"`
		}
		_ = json.Unmarshal(mgmt, &parsed)
		if len(parsed.Providers) == 0 {
			fmt.Println(entry.ProviderID, "no providers")
			continue
		}
		models, ok := parsed.Providers[0]["models"]
		if !ok {
			fmt.Println(entry.ProviderID, "no models key")
			continue
		}
		var asArray []map[string]json.RawMessage
		if err := json.Unmarshal(models, &asArray); err != nil {
			fmt.Println(entry.ProviderID, "array decode:", err)
			continue
		}
		if len(asArray) == 0 {
			fmt.Println(entry.ProviderID, "models empty")
			continue
		}
		types := map[string]string{}
		for key, value := range asArray[0] {
			var probe any
			_ = json.Unmarshal(value, &probe)
			switch probe.(type) {
			case string:
				types[key] = "string"
			case bool:
				types[key] = "bool"
			case float64:
				types[key] = "number"
			case []any:
				types[key] = "array"
			case map[string]any:
				types[key] = "object"
			default:
				types[key] = "null"
			}
		}
		fmt.Printf("%s models=%d firstTypes=%v\n", entry.ProviderID, len(asArray), types)
		if rawModel, ok := asArray[0]["model"]; ok {
			var modelObj map[string]json.RawMessage
			if err := json.Unmarshal(rawModel, &modelObj); err == nil {
				inner := map[string]string{}
				for key, value := range modelObj {
					var probe any
					_ = json.Unmarshal(value, &probe)
					switch probe.(type) {
					case string:
						inner[key] = "string"
					case bool:
						inner[key] = "bool"
					case float64:
						inner[key] = "number"
					case []any:
						inner[key] = "array"
					case map[string]any:
						inner[key] = "object"
					default:
						inner[key] = "null"
					}
				}
				fmt.Println("  model object types:", inner)
			}
		}
	}
}

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
