package metricmock

import (
	"context"

	"github.com/anyproto/any-sync/metric"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"storj.io/drpc"

	"github.com/anyproto/any-sync/app"
)

const CName = "common.metric"

func New() Metric {
	// TODO: Use real on after this issue is resolved
	// https://github.com/anyproto/any-sync/issues/373

	return &metricMock{}
}

type Metric interface {
	Registry() *prometheus.Registry
	WrapDRPCHandler(h drpc.Handler) drpc.Handler
	RequestLog(ctx context.Context, rpc string, fields ...zap.Field)
	RegisterSyncMetric(spaceId string, syncMetric metric.SyncMetric)
	UnregisterSyncMetric(spaceId string)
	RegisterStreamPoolSyncMetric(mtr metric.StreamPoolMetric)
	UnregisterStreamPoolSyncMetric()
	app.ComponentRunnable
}

type metricMock struct{}

func (m *metricMock) RequestLog(ctx context.Context, rpc string, fields ...zap.Field) {
	panic("implement me")
}

func (m *metricMock) RegisterStreamPoolSyncMetric(mtr metric.StreamPoolMetric) {
	panic("implement me")
}

func (m *metricMock) UnregisterStreamPoolSyncMetric() {
	panic("implement me")
}

func (m *metricMock) RegisterSyncMetric(spaceId string, syncMetric metric.SyncMetric) {
	panic("implement me")
}

func (m *metricMock) UnregisterSyncMetric(spaceId string) {
	panic("implement me")
}

func (m *metricMock) Init(a *app.App) (err error) {
	return nil
}

func (m *metricMock) Name() string {
	return CName
}

func (m *metricMock) Run(ctx context.Context) (err error) {
	return nil
}

func (m *metricMock) Registry() *prometheus.Registry {
	// Usually other code check on nil
	return nil
}

func (m *metricMock) WrapDRPCHandler(h drpc.Handler) drpc.Handler {
	// We just do not wrap
	return h
}

func (m *metricMock) Close(ctx context.Context) (err error) {
	return
}
