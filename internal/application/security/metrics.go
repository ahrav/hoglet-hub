package security

import "context"

// SecurityMetrics defines metrics for security monitoring.
type SecurityMetrics interface {
	// RecordActorActivity records activity by a specific actor.
	RecordActorActivity(ctx context.Context, actorID string, actorType string, activity string)

	// RecordIPAnomaly records anomalous IP activity.
	RecordIPAnomaly(ctx context.Context, ip string, anomalyType string, severity int)
}
