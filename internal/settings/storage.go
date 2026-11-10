package settings

import (
	"context"
	"fmt"
)

// StorageReading 是集成层从服务端回执映射出的原始读数。字段为 nil 表示服务端
// 没有给出该项；缺项不能被当成 0 字节。
type StorageReading struct {
	DatabaseBytes    *int64
	WALBytes         *int64
	ReclaimableBytes *int64
}

// StorageReport 是校验后的存储占用读数，单位是字节。它只有读数：本包没有清理、
// 删除或本地估算的入口；回收空间由离线指令完成。
type StorageReport struct {
	DatabaseBytes    int64
	WALBytes         int64
	ReclaimableBytes int64
}

// StorageReader 读取存储占用。实现方只调用服务端的存储读数方法，不做删除。
type StorageReader interface {
	ReadStorage(ctx context.Context) (StorageReading, error)
}

// RefreshStorage 读取一次存储占用。三个字节数必须都给出且非负；缺项或负数按
// 事实不合法拒绝，不返回零值读数。
func (m *Manager) RefreshStorage(ctx context.Context) (StorageReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.requireFeature(FeatureStorageReport); err != nil {
		return StorageReport{}, err
	}
	if m.cfg.Storage == nil {
		return StorageReport{}, missing(FeatureStorageReport, "未提供存储读数读取接口")
	}
	reading, err := m.cfg.Storage.ReadStorage(ctx)
	if err != nil {
		return StorageReport{}, readFailure(FeatureStorageReport, ErrStorageRead, err)
	}
	report, err := normalizeStorage(reading)
	if err != nil {
		return StorageReport{}, err
	}
	m.mu.Lock()
	m.storage = report
	m.storageOK = true
	m.mu.Unlock()
	return report, nil
}

// Storage 返回最近一次确认的存储读数；第二个返回值表示是否读过。
func (m *Manager) Storage() (StorageReport, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.storageOK {
		return StorageReport{}, false
	}
	return m.storage, true
}

func normalizeStorage(reading StorageReading) (StorageReport, error) {
	if reading.DatabaseBytes == nil {
		return StorageReport{}, fmt.Errorf("%w: 缺少数据库占用", ErrMalformedStorage)
	}
	if reading.WALBytes == nil {
		return StorageReport{}, fmt.Errorf("%w: 缺少 WAL 占用", ErrMalformedStorage)
	}
	if reading.ReclaimableBytes == nil {
		return StorageReport{}, fmt.Errorf("%w: 缺少可回收字节", ErrMalformedStorage)
	}
	report := StorageReport{
		DatabaseBytes:    *reading.DatabaseBytes,
		WALBytes:         *reading.WALBytes,
		ReclaimableBytes: *reading.ReclaimableBytes,
	}
	if report.DatabaseBytes < 0 {
		return StorageReport{}, fmt.Errorf("%w: 数据库占用为负 %d", ErrMalformedStorage, report.DatabaseBytes)
	}
	if report.WALBytes < 0 {
		return StorageReport{}, fmt.Errorf("%w: WAL 占用为负 %d", ErrMalformedStorage, report.WALBytes)
	}
	if report.ReclaimableBytes < 0 {
		return StorageReport{}, fmt.Errorf("%w: 可回收字节为负 %d", ErrMalformedStorage, report.ReclaimableBytes)
	}
	return report, nil
}
