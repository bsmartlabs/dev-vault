package secretsync

import "github.com/bsmartlabs/dev-vault/internal/mapping"

func runBatch[R any](
	targets []mapping.Target,
	operation string,
	runOne func(mapping.Target) (R, error),
	toFailure func(mapping.Target, error) BatchFailure,
) ([]R, []BatchFailure, BatchSummary) {
	succeeded := make([]R, 0, len(targets))
	failedItems := make([]BatchFailure, 0, len(targets))
	failed := 0
	for _, target := range targets {
		result, err := runOne(target)
		if err != nil {
			failedItems = append(failedItems, toFailure(target, err))
			failed++
		} else {
			succeeded = append(succeeded, result)
		}
	}
	return succeeded, failedItems, BatchSummary{
		Operation: operation,
		Failed:    failed,
		Total:     len(targets),
	}
}
