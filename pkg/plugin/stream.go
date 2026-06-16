package plugin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

func (ds *MQTTDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	// Extract the topic key from the channel path
	// Channel path format: "ds/{uid}/{topicKey}" where topicKey includes streaming key
	// We need to remove the channelPrefix ("ds/{uid}") to get the topic key
	topicKey := strings.TrimPrefix(req.Path, ds.channelPrefix+"/")
	logger := log.DefaultLogger.FromContext(ctx)

	chunks := strings.Split(topicKey, "/")
	if len(chunks) < 2 {
		return backend.DownstreamErrorf("invalid topic key: %s", topicKey)
	}

	interval, err := time.ParseDuration(chunks[0])
	if err != nil {
		return backend.DownstreamErrorf("invalid interval: %s", chunks[0])
	}

	_, err = ds.Client.Subscribe(topicKey, logger)
	if err != nil {
		return err
	}
	defer func() {
		if unsubErr := ds.Client.Unsubscribe(topicKey, logger); unsubErr != nil {
			logger.Error("Failed to unsubscribe from MQTT topic", "topicKey", topicKey, "error", unsubErr)
		}
	}()

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("stopped streaming (context canceled)", "path", req.Path, "topicKey", topicKey)
			ticker.Stop()
			return nil
		case <-ticker.C:
			topic, ok := ds.Client.GetTopic(topicKey)
			if !ok {
				logger.Debug("topic not found", "path", req.Path, "topicKey", topicKey)
				break
			}
			frame, err := topic.ToDataFrame(logger)
			if err != nil {
				logger.Error("failed to convert topic to data frame", "path", req.Path, "error", backend.DownstreamError(err))
				break
			}
			topic.Messages = []mqtt.Message{}
			if err := sender.SendFrame(frame, data.IncludeAll); err != nil {
				logger.Error("failed to send data frame", "path", req.Path, "error", backend.DownstreamError(err))
			}

		}
	}
}

func (ds *MQTTDatasource) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	// Extract orgId from the streaming key embedded in the channel path
	// Channel: {interval}/{topic}/{datasourceUid}/{hash}/{orgId}
	pathParts := strings.Split(req.Path, "/")
	if len(pathParts) < 5 {
		return &backend.SubscribeStreamResponse{
			Status: backend.SubscribeStreamStatusNotFound,
		}, backend.DownstreamErrorf("invalid channel path format")
	}

	orgId, err := strconv.ParseInt(pathParts[len(pathParts)-1], 10, 64)
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
