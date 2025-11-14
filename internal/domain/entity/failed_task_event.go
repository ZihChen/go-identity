package entity

import (
	"time"
)

// FailedTaskEvent 失敗任務事件實體
type FailedTaskEvent struct {
	id           uint
	taskID       string
	taskType     string
	queueName    string
	payload      string
	errorMessage string
	retryCount   int
	failedAt     time.Time
	redisKey     *string
	redisState   *string
	createdAt    time.Time
	updatedAt    time.Time
}

// NewFailedTaskEvent 創建新的失敗任務事件（使用當前時間）
func NewFailedTaskEvent(taskID, taskType, queueName, payload, errorMessage string, retryCount int) *FailedTaskEvent {
	now := time.Now()
	return &FailedTaskEvent{
		taskID:       taskID,
		taskType:     taskType,
		queueName:    queueName,
		payload:      payload,
		errorMessage: errorMessage,
		retryCount:   retryCount,
		failedAt:     now,
		createdAt:    now,
		updatedAt:    now,
	}
}

// NewFailedTaskEventWithTimes 創建新的失敗任務事件（指定時間）
func NewFailedTaskEventWithTimes(taskID, taskType, queueName, payload, errorMessage string, retryCount int, failedAt, createdAt, updatedAt time.Time) *FailedTaskEvent {
	return &FailedTaskEvent{
		taskID:       taskID,
		taskType:     taskType,
		queueName:    queueName,
		payload:      payload,
		errorMessage: errorMessage,
		retryCount:   retryCount,
		failedAt:     failedAt,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

// Getter 方法
func (f *FailedTaskEvent) GetID() uint {
	return f.id
}

func (f *FailedTaskEvent) GetTaskID() string {
	return f.taskID
}

func (f *FailedTaskEvent) GetTaskType() string {
	return f.taskType
}

func (f *FailedTaskEvent) GetQueueName() string {
	return f.queueName
}

func (f *FailedTaskEvent) GetPayload() string {
	return f.payload
}

func (f *FailedTaskEvent) GetErrorMessage() string {
	return f.errorMessage
}

func (f *FailedTaskEvent) GetRetryCount() int {
	return f.retryCount
}

func (f *FailedTaskEvent) GetFailedAt() time.Time {
	return f.failedAt
}

func (f *FailedTaskEvent) GetRedisKey() *string {
	return f.redisKey
}

func (f *FailedTaskEvent) GetRedisState() *string {
	return f.redisState
}

func (f *FailedTaskEvent) GetCreatedAt() time.Time {
	return f.createdAt
}

func (f *FailedTaskEvent) GetUpdatedAt() time.Time {
	return f.updatedAt
}

// Setter 方法
func (f *FailedTaskEvent) SetID(id uint) {
	f.id = id
}

func (f *FailedTaskEvent) SetTaskID(taskID string) {
	f.taskID = taskID
}

func (f *FailedTaskEvent) SetTaskType(taskType string) {
	f.taskType = taskType
}

func (f *FailedTaskEvent) SetQueueName(queueName string) {
	f.queueName = queueName
}

func (f *FailedTaskEvent) SetPayload(payload string) {
	f.payload = payload
}

func (f *FailedTaskEvent) SetErrorMessage(errorMessage string) {
	f.errorMessage = errorMessage
}

func (f *FailedTaskEvent) SetRetryCount(retryCount int) {
	f.retryCount = retryCount
}

func (f *FailedTaskEvent) SetFailedAt(failedAt time.Time) {
	f.failedAt = failedAt
}

func (f *FailedTaskEvent) SetRedisKey(redisKey *string) {
	f.redisKey = redisKey
}

func (f *FailedTaskEvent) SetRedisState(redisState *string) {
	f.redisState = redisState
}

func (f *FailedTaskEvent) SetCreatedAt(createdAt time.Time) {
	f.createdAt = createdAt
}

func (f *FailedTaskEvent) SetUpdatedAt(updatedAt time.Time) {
	f.updatedAt = updatedAt
}

// IsValid 驗證失敗任務事件是否有效
func (f *FailedTaskEvent) IsValid() bool {
	return f.taskID != "" && f.taskType != "" && f.queueName != "" && f.errorMessage != ""
}