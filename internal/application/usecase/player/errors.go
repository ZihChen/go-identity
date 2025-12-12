package usecase

import "errors"

var (
	// ErrBatchProcessorFull 批次處理器已滿
	ErrBatchProcessorFull = errors.New("batch processor channel is full")

	// ErrBatchProcessorNotStarted 批次處理器未啟動
	ErrBatchProcessorNotStarted = errors.New("batch processor is not started")
)
