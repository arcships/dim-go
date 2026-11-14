package uiwire

import (
	"encoding/json"

	"github.com/arcships/dimagent-mygo/internal/interaction"
)

// approvalResponse 把界面上的决定换成应答：允许一次 / 一直允许 / 拒绝。
func approvalResponse(decision string) interaction.Response {
	switch decision {
	case "allow-always":
		return interaction.Response{Decision: interaction.DecisionAllowLongTerm}
	case "deny":
		return interaction.Response{Decision: interaction.DecisionReject}
	default:
		return interaction.Response{Decision: interaction.DecisionApprove}
	}
}

// questionResponse 把回答包成应答。
func questionResponse(answer string) interaction.Response {
	raw, _ := json.Marshal(map[string]string{"answer": answer})
	return interaction.Response{Decision: interaction.DecisionAnswer, Answer: raw}
}
