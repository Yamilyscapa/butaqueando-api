package health

import (
	"context"
	"time"

	"gorm.io/gorm"
)

const defaultPingTimeout = 2 * time.Second

type Service struct {
	db          *gorm.DB
	pingTimeout time.Duration
}

type CheckResult struct {
	Ready    bool
	Database bool
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db:          db,
		pingTimeout: defaultPingTimeout,
	}
}

func (s *Service) Check(ctx context.Context) CheckResult {
	if s == nil || s.db == nil {
		return CheckResult{Ready: false, Database: false}
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		return CheckResult{Ready: false, Database: false}
	}

	pingCtx, cancel := context.WithTimeout(ctx, s.pingTimeout)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return CheckResult{Ready: false, Database: false}
	}

	return CheckResult{Ready: true, Database: true}
}
