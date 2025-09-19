package db

import (
	"context"
)

// StubTxManager는 트랜잭션 없이 함수를 직접 실행하는 테스트용 구현체입니다.
// 트랜잭션 로직이 중요하지 않은 단위 테스트에서 사용됩니다.
type StubTxManager struct{}

// NewStubTxManager는 트랜잭션을 시뮬레이션하지 않고 함수를 직접 실행하는 TxManager를 반환합니다.
func NewStubTxManager() TxManager {
	return &StubTxManager{}
}

// RunInTx는 트랜잭션 없이 제공된 함수를 직접 실행합니다.
func (s *StubTxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if fn == nil {
		return nil
	}
	return fn(ctx)
}
