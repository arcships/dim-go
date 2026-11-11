package projection

// 证据用例的公共环境：临时目录只从环境变量来，测试里不写死任何本机路径。

import (
	"os"
	"testing"
)

// evidenceTempDir 建一个本次运行专用的临时目录。
//
// 默认用系统临时目录；daemon 的 Unix 端点有长度上限，路径太长时用 DIM_EVIDENCE_TMP
// 指一个更短的临时根。两者都不写死在代码里。
func evidenceTempDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp(os.Getenv("DIM_EVIDENCE_TMP"), prefix)
	if err != nil {
		t.Skipf("无法创建临时目录（可用 DIM_EVIDENCE_TMP 指定临时根）: %v", err)
	}
	// 端点会接成 <dir>/home/rust/daemon.sock，超过上界就先说清楚，而不是连不上。
	if len(dir)+len("/home/rust/daemon.sock") > 100 {
		t.Skipf("临时目录路径过长（%d 字节，端点会超出上限）；用 DIM_EVIDENCE_TMP 指定更短的临时根", len(dir))
	}
	return dir
}
