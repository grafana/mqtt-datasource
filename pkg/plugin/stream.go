package plugin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func (ds *MQTTDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	logger := log.DefaultLogger.FromContext(ctx)

	chunks := strings.Split(req.Path, "/")

	// Expected format: {interval}/{encodedTopic}/{dsUid}/{hash}/{orgId}/{refId}
	if len(chunks) < 6 {
		return backend.DownstreamErrorf("invalid topic key: %s", req.Path)
	}

	refID := chunks[len(chunks)-1]

	interval, err := time.ParseDuration(chunks[0])
	if err != nil {
		return backend.DownstreamErrorf("invalid interval: %s", chunks[0])
	}

	_, err = ds.Client.Subscribe(req.Path, logger)
	if err != nil {
		return err
	}
	defer func() {
		if unsubErr := ds.Client.Unsubscribe(req.Path, logger); unsubErr != nil {
			logger.Error("Failed to unsubscribe from MQTT topic", "path", req.Path, "error", unsubErr)
		}
	}()

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("stopped streaming (context canceled)", "path", req.Path)
			ticker.Stop()
			return nil
		case <-ticker.C:
			topic, ok := ds.Client.GetTopic(req.Path)
			if !ok {
				logger.Debug("topic not found", "path", req.Path)
				break
			}
			frame, err := topic.ToDataFrame(logger)
			if err != nil {
				logger.Error("failed to convert topic to data frame", "path", req.Path, "error", backend.DownstreamError(err))
				break
			}
			frame.Name = refID
			topic.KeepLastMessage()
			if err := sender.SendFrame(frame, data.IncludeAll); err != nil {
				logger.Error("failed to send data frame", "path", req.Path, "error", backend.DownstreamError(err))
			}

		}
	}
}

func (ds *MQTTDatasource) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	// Extract orgId from the streaming key embedded in the channel path
	// Channel: {interval}/{topic}/{datasourceUid}/{hash}/{orgId}/{refId}
	pathParts := strings.Split(req.Path, "/")
	if len(pathParts) < 6 {
		return &backend.SubscribeStreamResponse{
			Status: backend.SubscribeStreamStatusNotFound,
		}, backend.DownstreamErrorf("invalid channel path format")
	}

	orgId, err := strconv.ParseInt(pathParts[len(pathParts)-2], 10, 64)
	if err != nil {
		return &backend.SubscribeStreamResponse{
			Status: backend.SubscribeStreamStatusNotFound,
		}, backend.DownstreamErrorf("unable to determine orgId from request")
	}

	pluginCfg := backend.PluginConfigFromContext(ctx)
	if orgId != pluginCfg.OrgID {
		return &backend.SubscribeStreamResponse{
			Status: backend.SubscribeStreamStatusPermissionDenied,
		}, backend.DownstreamErrorf("invalid orgId supplied in request")
	}

	return &backend.SubscribeStreamResponse{
		Status: backend.SubscribeStreamStatusOK,
	}, nil
}

func (ds *MQTTDatasource) PublishStream(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}
