package core

import (
	"fmt"
	"strings"

	dto "github.com/prometheus/client_model/go"
)

type DisallowCertainMetricsChecker struct {
	disallowedMetrics []string
}

func (d DisallowCertainMetricsChecker) String() string {
	return fmt.Sprintf("DisallowCertainMetricsChecker{disallowedMetrics: %v}", d.disallowedMetrics)
}

func NewDisallowCertainMetricsChecker(metrics []string) *DisallowCertainMetricsChecker {
	return &DisallowCertainMetricsChecker{
		disallowedMetrics: metrics,
	}
}

func (c *DisallowCertainMetricsChecker) Check(metricFamilies map[string]*dto.MetricFamily) (bool, string) {
	for _, metric := range c.disallowedMetrics {
		if _, ok := metricFamilies[metric]; ok {
			return false, fmt.Sprintf("metric %s is disallowed but was found", metric)
		}
	}
	return true, ""
}

type DisallowEmptyMetricsChecker struct{}

func (d DisallowEmptyMetricsChecker) String() string {
	return "DisallowEmptyMetricsChecker"
}

func NewDisallowEmptyMetricsChecker() *DisallowEmptyMetricsChecker {
	return &DisallowEmptyMetricsChecker{}
}

func (c *DisallowEmptyMetricsChecker) Check(metricFamilies map[string]*dto.MetricFamily) (bool, string) {
	if len(metricFamilies) == 0 {
		return false, "metricFamilies should not be empty"
	}
	return true, ""
}

type SingleMetricExistsChecker struct {
	expectedMetric string
}

func (s SingleMetricExistsChecker) String() string {
	return fmt.Sprintf("SingleMetricExistsChecker{expectedMetric: %s}", s.expectedMetric)
}

func NewSingleMetricExistsChecker(expectedMetric string) *SingleMetricExistsChecker {
	return &SingleMetricExistsChecker{
		expectedMetric: expectedMetric,
	}
}

func (c *SingleMetricExistsChecker) Check(metricFamilies map[string]*dto.MetricFamily) (bool, string) {
	_, ok := metricFamilies[c.expectedMetric]
	if !ok {
		return false, fmt.Sprintf("expected metric %s is missing", c.expectedMetric)
	}
	return true, ""
}

type SingleMetricTypeChecker struct {
	expectedMetric string
	expectedType   string
}

func (s SingleMetricTypeChecker) String() string {
	return fmt.Sprintf("SingleMetricTypeChecker{expectedMetric: %s, expectedType: %s}", s.expectedMetric, s.expectedType)
}

func NewSingleMetricTypeChecker(expectedMetric string, expectedType string) *SingleMetricTypeChecker {
	return &SingleMetricTypeChecker{
		expectedMetric: expectedMetric,
		expectedType:   expectedType,
	}
}

func (c *SingleMetricTypeChecker) Check(metricFamilies map[string]*dto.MetricFamily) (bool, string) {
	metricFamily, ok := metricFamilies[c.expectedMetric]
	if !ok {
		return false, fmt.Sprintf("expected metric %s is missing", c.expectedMetric)
	}
	metricType := metricFamily.GetType()
	if metricType.String() != strings.ToUpper(c.expectedType) {
		return false, fmt.Sprintf("expected metric %s should be of type %s but was %s", c.expectedMetric, c.expectedType, metricFamily.GetType())
	}
	return true, ""
}

type MetricLabelChecker struct {
	expectedMetric string
	expectedLabels []string
}

func (m MetricLabelChecker) String() string {
	return fmt.Sprintf("MetricLabelChecker{expectedMetric: %s, expectedLabels: %v}", m.expectedMetric, m.expectedLabels)
}

func NewMetricLabelChecker(expectedMetric string, expectedLabels []string) *MetricLabelChecker {
	return &MetricLabelChecker{
		expectedMetric: expectedMetric,
		expectedLabels: expectedLabels,
	}
}

func (c *MetricLabelChecker) Check(metricFamilies map[string]*dto.MetricFamily) (bool, string) {
	metricFamily, ok := metricFamilies[c.expectedMetric]
	if !ok {
		return false, fmt.Sprintf("expected metric %s is missing", c.expectedMetric)
	}

	for _, metric := range metricFamily.GetMetric() {
		labelNames := make(map[string]bool)
		for _, label := range metric.GetLabel() {
			labelNames[label.GetName()] = true
		}

		for _, expectedLabel := range c.expectedLabels {
			if _, ok := labelNames[expectedLabel]; !ok {
				return false, fmt.Sprintf("expected label %s is missing in metric %s", expectedLabel, c.expectedMetric)
			}
		}
	}
	return true, ""
}

type MetricLabelDisallowChecker struct {
	expectedMetric   string
	disallowedLabels []string
}

func NewMetricLabelDisallowChecker(expectedMetric string, disallowedLabels []string) *MetricLabelDisallowChecker {
	return &MetricLabelDisallowChecker{
		expectedMetric:   expectedMetric,
		disallowedLabels: disallowedLabels,
	}
}

func (c *MetricLabelDisallowChecker) Check(metricFamilies map[string]*dto.MetricFamily) (bool, string) {
	metricFamily, ok := metricFamilies[c.expectedMetric]
	if !ok {
		return false, fmt.Sprintf("expected metric %s is missing", c.expectedMetric)
	}

	for _, metric := range metricFamily.GetMetric() {
		labelNames := make(map[string]bool)
		for _, label := range metric.GetLabel() {
			labelNames[label.GetName()] = true
		}

		for _, disallowedLabel := range c.disallowedLabels {
			if _, ok := labelNames[disallowedLabel]; ok {
				return false, fmt.Sprintf("disallowed label %s is present in metric %s", disallowedLabel, c.expectedMetric)
			}
		}
	}
	return true, ""
}
