package templates

import "regexp"

// AnalyticsMeasurementID is injected by the release image build from the
// repository's GA_MEASUREMENT_ID variable. Empty local/test builds omit GA4.
var AnalyticsMeasurementID string

var measurementIDPattern = regexp.MustCompile(`^G-[A-Z0-9]+$`)

func analyticsEnabled() bool {
	return measurementIDPattern.MatchString(AnalyticsMeasurementID)
}
