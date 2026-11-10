package app

import (
	"context"
	"fmt"
	"time"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// 账号领域在服务端上的读取方法。形状取自账号领域的集成证据。
const (
	methodAccountStatus = "account.status"
	methodAccountUsage  = "account.usage"
	methodModelList     = "model.list"
	methodLoginBegin    = "account.login.begin"
	methodLoginComplete = "account.login.complete"
)

// accountStatusReader 读取账号状态。
type accountStatusReader struct {
	app *App
}

func (r *accountStatusReader) ReadAccount(ctx context.Context) (account.Fact, error) {
	client, err := r.app.requireControl("读取账号状态")
	if err != nil {
		return account.Fact{}, err
	}
	var reply struct {
		Status   string `json:"status"`
		Identity struct {
			AccountID   string `json:"accountId"`
			Email       string `json:"email"`
			DisplayName string `json:"displayName"`
			Plan        string `json:"plan"`
		} `json:"identity"`
	}
	if err := client.CallInto(ctx, methodAccountStatus, nil, &reply); err != nil {
		return account.Fact{}, err
	}
	status, err := parseAccountStatus(reply.Status)
	if err != nil {
		return account.Fact{}, err
	}
	return account.Fact{
		Status: status,
		Identity: account.Identity{
			AccountID:   reply.Identity.AccountID,
			Email:       reply.Identity.Email,
			DisplayName: reply.Identity.DisplayName,
			Plan:        reply.Identity.Plan,
		},
	}, nil
}

func parseAccountStatus(raw string) (account.Status, error) {
	switch account.Status(raw) {
	case account.StatusSignedIn, account.StatusSignedOut, account.StatusExpired:
		return account.Status(raw), nil
	default:
		return "", fmt.Errorf("%w: 账号状态无法识别 %q", account.ErrMalformedFact, raw)
	}
}

// accountUsageReader 逐项读取用量；聚合项只在服务端给出时可用。
type accountUsageReader struct {
	app *App
}

func (r *accountUsageReader) ReadUsage(ctx context.Context) (account.Usage, error) {
	client, err := r.app.requireControl("读取账号用量")
	if err != nil {
		return account.Usage{}, err
	}
	var reply struct {
		Lines []struct {
			Key      string  `json:"key"`
			Label    string  `json:"label"`
			Used     float64 `json:"used"`
			Limit    float64 `json:"limit"`
			Unit     string  `json:"unit"`
			Window   string  `json:"window"`
			ResetsAt int64   `json:"resetsAt"`
		} `json:"lines"`
		Aggregate *struct {
			Used  float64 `json:"used"`
			Limit float64 `json:"limit"`
			Unit  string  `json:"unit"`
		} `json:"aggregate"`
	}
	if err := client.CallInto(ctx, methodAccountUsage, nil, &reply); err != nil {
		return account.Usage{}, err
	}
	usage := account.Usage{Lines: make([]account.UsageLine, 0, len(reply.Lines))}
	for _, line := range reply.Lines {
		mapped := account.UsageLine{
			Key:    line.Key,
			Label:  line.Label,
			Used:   line.Used,
			Limit:  line.Limit,
			Unit:   line.Unit,
			Window: line.Window,
		}
		if line.ResetsAt > 0 {
			mapped.ResetsAt = time.UnixMilli(line.ResetsAt).UTC()
		}
		usage.Lines = append(usage.Lines, mapped)
	}
	if reply.Aggregate != nil {
		usage.Aggregate = account.AggregateUsage{
			Available: true,
			Used:      reply.Aggregate.Used,
			Limit:     reply.Aggregate.Limit,
			Unit:      reply.Aggregate.Unit,
		}
	}
	return usage, nil
}

// accountModelReader 读取服务端声明的模型清单。
type accountModelReader struct {
	app *App
}

func (r *accountModelReader) ReadModels(ctx context.Context) ([]account.Model, error) {
	client, err := r.app.requireControl("读取模型清单")
	if err != nil {
		return nil, err
	}
	var reply struct {
		Models []struct {
			ID              string `json:"id"`
			DisplayName     string `json:"displayName"`
			RequiresAccount bool   `json:"requiresAccount"`
			Unavailable     bool   `json:"unavailable"`
		} `json:"models"`
	}
	if err := client.CallInto(ctx, methodModelList, nil, &reply); err != nil {
		return nil, err
	}
	models := make([]account.Model, 0, len(reply.Models))
	for _, model := range reply.Models {
		models = append(models, account.Model{
			ID:              model.ID,
			DisplayName:     model.DisplayName,
			RequiresAccount: model.RequiresAccount,
			Unavailable:     model.Unavailable,
		})
	}
	return models, nil
}

// accountAuthSource 是浏览器登录的服务端侧：发起授权与用回调结果完成交换。
type accountAuthSource struct {
	app *App
}

func (s *accountAuthSource) BeginAuthorization(
	ctx context.Context,
	redirectURI string,
) (account.AuthorizationRequest, error) {
	client, err := s.app.requireControl("发起账号登录")
	if err != nil {
		return account.AuthorizationRequest{}, err
	}
	var reply struct {
		URL   string `json:"url"`
		State string `json:"state"`
	}
	if err := client.CallInto(ctx, methodLoginBegin, map[string]any{"redirectUri": redirectURI}, &reply); err != nil {
		return account.AuthorizationRequest{}, err
	}
	return account.AuthorizationRequest{URL: reply.URL, State: reply.State}, nil
}

func (s *accountAuthSource) CompleteAuthorization(
	ctx context.Context,
	result account.AuthorizationResult,
) error {
	client, err := s.app.requireControl("完成账号登录")
	if err != nil {
		return err
	}
	_, err = client.Call(ctx, methodLoginComplete, map[string]any{
		"code":  result.Code,
		"state": result.State,
	})
	return err
}

// requireControl 返回控制连接；未连接时给出连接缺失分类，调用方据此失败关闭。
func (a *App) requireControl(op string) (*protocol.Client, error) {
	client := a.control()
	if client == nil {
		return nil, failure.New(failure.KindConnectionMissing, op, ErrNotConnected)
	}
	return client, nil
}
