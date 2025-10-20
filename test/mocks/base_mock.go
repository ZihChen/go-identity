package mocks

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

// BaseMock 統一的 Mock 基礎結構
type BaseMock struct {
	mock.Mock
	t *testing.T
}

// NewBaseMock 創建新的基礎 Mock
func NewBaseMock(t *testing.T) *BaseMock {
	return &BaseMock{t: t}
}

// AssertExpectations 驗證所有期望調用
func (m *BaseMock) AssertExpectations() {
	if m.t != nil {
		m.Mock.AssertExpectations(m.t)
	}
}
