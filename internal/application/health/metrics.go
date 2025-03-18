package health

import "context"

type HealthMetrics interface {
	SetSystemHealth(ctx context.Context, status bool)
}
