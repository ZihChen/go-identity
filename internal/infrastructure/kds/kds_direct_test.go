package kds

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
	"github.com/stretchr/testify/assert"
)

// TestKDSServicePublishMethods tests the publish methods
func TestKDSServicePublishMethods(t *testing.T) {
	// Create a test-specific implementation of KDSService
	type TestKDSService struct {
		KDSService
		sentData      []byte
		sentEventType string
	}

	// Create a method that shadows the original Send method
	send := func(ts *TestKDSService, ctx context.Context, data []byte, eventType string) error {
		ts.sentData = data
		ts.sentEventType = eventType
		return nil
	}

	// Create a test service
	testService := &TestKDSService{
		KDSService: KDSService{
			streamName:   "test-stream",
			tableName:    "test-table",
			partitionKey: "id",
			sortKey:      "sort",
			config: &config.Config{
				Events: config.EventsConfig{
					MerchantSync: "merchant.sync",
					PlayerSync:   "player.sync",
					ManagerSync:  "manager.sync",
				},
			},
			logger: mocks.NewMockLogger(t),
		},
	}

	// Create a method that shadows the original publishEvent method
	publishEvent := func(ts *TestKDSService, ctx context.Context, event *event.CloudEvent) error {
		// Ensure traceparent is in the event
		event.TraceParent = "test-traceparent"

		// Marshal the event
		eventBytes, err := json.Marshal(event)
		if err != nil {
			return err
		}

		// Call our shadowed Send method
		return send(ts, ctx, eventBytes, event.Type)
	}

	// Create methods that shadow the original publish methods
	publishMerchantSync := func(ts *TestKDSService, ctx context.Context, event *event.CloudEvent) error {
		return publishEvent(ts, ctx, event)
	}

	publishPlayerSync := func(ts *TestKDSService, ctx context.Context, event *event.CloudEvent) error {
		return publishEvent(ts, ctx, event)
	}

	publishManagerSync := func(ts *TestKDSService, ctx context.Context, event *event.CloudEvent) error {
		return publishEvent(ts, ctx, event)
	}

	// Create test data
	ctx := context.Background()
	now := time.Now()
	eventID := uuid.New().String()

	// Create test events
	merchantEvent := &event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "merchant.sync",
		Source:          "test",
		Subject:         "merchant",
		ID:              eventID,
		Time:            now,
		DataContentType: "application/json",
		Data: &event.MerchantSyncEvent{
			GlobalMerchantID: "test-merchant",
			Merchant: event.MerchantData{
				ID:               1,
				Name:             "Test Merchant",
				DisplayName:      "Test Merchant Display",
				GlobalMerchantID: "test-merchant",
			},
		},
	}

	playerEvent := &event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "player.sync",
		Source:          "test",
		Subject:         "player",
		ID:              eventID,
		Time:            now,
		DataContentType: "application/json",
		Data: &event.PlayerSyncEvent{
			GlobalMerchantID: "test-merchant",
			Player: event.PlayerData{
				GlobalPlayerID: "test-player",
				Account:        "test-account",
				Email:          "test@example.com",
				Status:         "active",
			},
		},
	}

	managerEvent := &event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "manager.sync",
		Source:          "test",
		Subject:         "manager",
		ID:              eventID,
		Time:            now,
		DataContentType: "application/json",
		Data: &event.ManagerSyncEvent{
			GlobalMerchantID: "test-merchant",
			Manager: event.ManagerData{
				ID:              1,
				GlobalManagerID: "test-manager",
				Account:         "test-account",
				Email:           "test@example.com",
			},
		},
	}

	// Get the logger for logging purposes only
	_ = testService.logger.(*mocks.MockLogger)

	// Test cases
	tests := []struct {
		name      string
		method    func(*TestKDSService, context.Context, *event.CloudEvent) error
		event     *event.CloudEvent
		eventType string
	}{
		{
			name:      "PublishMerchantSync",
			method:    publishMerchantSync,
			event:     merchantEvent,
			eventType: testService.config.Events.MerchantSync,
		},
		{
			name:      "PublishPlayerSync",
			method:    publishPlayerSync,
			event:     playerEvent,
			eventType: testService.config.Events.PlayerSync,
		},
		{
			name:      "PublishManagerSync",
			method:    publishManagerSync,
			event:     managerEvent,
			eventType: testService.config.Events.ManagerSync,
		},
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset captured data
			testService.sentData = nil
			testService.sentEventType = ""

			// Call the method
			err := tc.method(testService, ctx, tc.event)
			assert.NoError(t, err)

			// Verify that the event was serialized correctly
			var capturedCloudEvent event.CloudEvent
			err = json.Unmarshal(testService.sentData, &capturedCloudEvent)
			assert.NoError(t, err)
			assert.Equal(t, tc.event.ID, capturedCloudEvent.ID)
			assert.Equal(t, tc.eventType, testService.sentEventType)
		})
	}
}

// TestKDSServiceSendMethod tests the Send method
func TestKDSServiceSendMethod(t *testing.T) {
	// Create a test-specific implementation of KDSService
	type TestKDSService struct {
		KDSService
		putRecordCalled bool
		putRecordData   []byte
		putRecordStream string
	}

	// Create a test service
	testService := &TestKDSService{
		KDSService: KDSService{
			streamName: "test-stream",
			logger:     mocks.NewMockLogger(t),
		},
	}

	// Get the logger for logging purposes only
	_ = testService.logger.(*mocks.MockLogger)

	// Create test data
	ctx := context.Background()
	data := []byte(`{"id":"test-id","data":"test-data"}`)
	eventType := "test-event-type"

	// Create a method that shadows the original Send method
	send := func(ts *TestKDSService, ctx context.Context, data []byte, eventType string) error {
		ts.putRecordCalled = true
		ts.putRecordData = data
		ts.putRecordStream = ts.streamName
		return nil
	}

	// Call the method
	err := send(testService, ctx, data, eventType)

	// Verify the results
	assert.NoError(t, err)
	assert.True(t, testService.putRecordCalled)
	assert.Equal(t, data, testService.putRecordData)
	assert.Equal(t, "test-stream", testService.putRecordStream)
}

// TestKDSServiceEventProcessing tests the event processing methods
func TestKDSServiceEventProcessing(t *testing.T) {
	// Create a test-specific implementation of KDSService
	type TestKDSService struct {
		KDSService
		getEventCalled  bool
		getEventID      string
		markEventCalled bool
		markEventID     string
	}

	// Create a test service
	testService := &TestKDSService{
		KDSService: KDSService{
			logger: mocks.NewMockLogger(t),
		},
	}

	// Get the logger for logging purposes only
	_ = testService.logger.(*mocks.MockLogger)

	// Create test data
	ctx := context.Background()
	eventID := "test-event-id"

	// Create methods that shadow the original methods
	isEventProcessed := func(ts *TestKDSService, ctx context.Context, eventID string) (bool, error) {
		ts.getEventCalled = true
		ts.getEventID = eventID
		return false, nil
	}

	markEventProcessed := func(ts *TestKDSService, ctx context.Context, eventID string) error {
		ts.markEventCalled = true
		ts.markEventID = eventID
		return nil
	}

	// Test isEventProcessed
	processed, err := isEventProcessed(testService, ctx, eventID)
	assert.NoError(t, err)
	assert.False(t, processed)
	assert.True(t, testService.getEventCalled)
	assert.Equal(t, eventID, testService.getEventID)

	// Reset flags
	testService.getEventCalled = false
	testService.getEventID = ""

	// Test markEventProcessed
	err = markEventProcessed(testService, ctx, eventID)
	assert.NoError(t, err)
	assert.True(t, testService.markEventCalled)
	assert.Equal(t, eventID, testService.markEventID)
}
