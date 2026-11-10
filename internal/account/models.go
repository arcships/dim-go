package account

import (
	"context"
	"fmt"
	"strings"
)

// Model 是服务端声明的一个模型。可用性由服务端给出；本包不探测、不猜测。
type Model struct {
	// ID 是模型标识，必填。
	ID string
	// DisplayName 是展示名；空表示服务端没有给出。
	DisplayName string
	// RequiresAccount 表示这个模型需要账号可用；未登录或登录过期时不可选。
	RequiresAccount bool
	// Unavailable 表示服务端把该模型标为不可用；这类模型一律不出现在清单里。
	Unavailable bool
}

// ModelList 是按当前账号状态过滤后的可用模型清单。
type ModelList struct {
	// Models 是当前可用的模型，顺序与服务端一致。
	Models []Model
	// Ready 表示清单来自一次成功的服务端读取。
	Ready bool
	// Reason 说明清单为空或不全的原因；空表示服务端完整清单。
	Reason string
}

// RefreshModels 读取服务端模型清单，并按当前账号状态过滤。
//
// 结构不合法（缺标识或标识重复）时整份不采用，返回错误；不会只保留一半。
func (a *Account) RefreshModels(ctx context.Context) (ModelList, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if a.cfg.Models == nil {
		a.mu.Lock()
		a.modelsErr = ErrModelsUnavailable.Error()
		list := a.modelsLocked()
		a.mu.Unlock()
		return list, ErrModelsUnavailable
	}

	models, err := a.cfg.Models.ReadModels(ctx)
	if err != nil {
		a.mu.Lock()
		a.modelsErr = err.Error()
		list := a.modelsLocked()
		a.mu.Unlock()
		return list, fmt.Errorf("%w: %v", ErrModelsRead, err)
	}
	if err := validateModels(models); err != nil {
		a.mu.Lock()
		a.modelsErr = err.Error()
		list := a.modelsLocked()
		a.mu.Unlock()
		return list, err
	}

	a.mu.Lock()
	a.models = append([]Model(nil), models...)
	a.modelsOK = true
	a.modelsErr = ""
	list := a.modelsLocked()
	a.mu.Unlock()
	return list, nil
}

// Models 返回按当前账号状态过滤的可用模型清单。
//
// 过滤规则：
//   - 尚未读取：空清单，原因说明尚未读取；
//   - 未登录 / 状态未知：空清单，原因说明需要账号；
//   - 登录过期：只保留不需要账号的模型，并说明需要账号的模型不可用；
//   - 已登录：去掉服务端标为不可用的模型。
func (a *Account) Models() ModelList {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.modelsLocked()
}

func (a *Account) modelsLocked() ModelList {
	list := ModelList{Ready: a.modelsOK}

	// 账号不可用时根本谈不上可用模型：原因只谈账号状态，不拿「清单尚未读取」掩盖。
	switch a.status {
	case StatusUnknown:
		list.Reason = "账号状态未知：不能确定可用模型"
		return list
	case StatusSignedOut:
		list.Reason = "未登录：可用模型需要账号"
		return list
	}
	if !a.modelsOK {
		if a.status == StatusExpired {
			list.Reason = "登录已过期：可用模型需要重新登录"
			return list
		}
		if a.modelsErr != "" {
			list.Reason = a.modelsErr
		} else {
			list.Reason = "模型清单尚未读取"
		}
		return list
	}

	available := make([]Model, 0, len(a.models))
	for _, model := range a.models {
		if model.Unavailable {
			continue
		}
		available = append(available, model)
	}

	switch a.status {
	case StatusSignedIn:
		list.Models = available
		if a.modelsErr != "" {
			list.Reason = "最近一次模型清单读取失败：" + a.modelsErr
		}
		if len(available) == 0 && list.Reason == "" {
			list.Reason = "服务端没有声明可用模型"
		}
	case StatusExpired:
		filtered := make([]Model, 0, len(available))
		trimmed := false
		for _, model := range available {
			if model.RequiresAccount {
				trimmed = true
				continue
			}
			filtered = append(filtered, model)
		}
		list.Models = filtered
		switch {
		case trimmed && len(filtered) == 0:
			list.Reason = "登录已过期：没有不需要账号的可用模型"
		case trimmed:
			list.Reason = "登录已过期：需要账号的模型不可用"
		case len(filtered) == 0:
			list.Reason = "服务端没有声明可用模型"
		}
	}
	return list
}

// validateModels 校验模型清单的结构：标识必填且不重复。重复标识无法稳定选择，
// 按不合法处理而不是悄悄保留一条。
func validateModels(models []Model) error {
	seen := make(map[string]bool, len(models))
	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			return fmt.Errorf("%w: 模型缺少标识", ErrMalformedModels)
		}
		if seen[id] {
			return fmt.Errorf("%w: 模型标识重复 %q", ErrMalformedModels, id)
		}
		seen[id] = true
	}
	return nil
}
