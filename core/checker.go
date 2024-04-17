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

func (m MetricLabelDisallowChecker) String() string {
	return fmt.Sprintf("MetricLabelDisallowChecker{expectedMetric: %s, disallowedLabels: %v}", m.expectedMetric, m.disallowedLabels)
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

type metricFilter struct {
	Labels map[string]string
}

func (f *metricFilter) isMetricMatch(metric *dto.Metric) bool {
	matchedLabels := 0

	for _, label := range metric.GetLabel() {
		value, ok := f.Labels[label.GetName()]
		if ok && value != label.GetValue() {
			return false
		}

		matchedLabels++
	}

	return matchedLabels == len(f.Labels)
}

func newMetricFilter(labels map[string]string) *metricFilter {
	return &metricFilter{
		Labels: labels,
	}
}

type MetricSampleChecker struct {
	*metricFilter
	Name string
}

func (m *MetricSampleChecker) String() string {
	return fmt.Sprintf("MetricSampleChecker{Labels: %v}", m.Labels)
}

func (m *MetricSampleChecker) Check(metricFamilies map[string]*dto.MetricFamily) (bool, string) {
	for _, metricFamily := range metricFamilies {
		if metricFamily.GetName() != m.Name {
			continue
		}

		for _, metric := range metricFamily.GetMetric() {
			if m.isMetricMatch(metric) {
				return true, ""
			}
		}
	}

	return false, fmt.Sprintf("expected sample not found in metric %s", m.Name)
}

func NewMetricSampleChecker(name string, labels map[string]string) *MetricSampleChecker {
	return &MetricSampleChecker{
		metricFilter: newMetricFilter(labels),
		Name:         name,
	}
}

type MetricSampleValueChecker struct {
	*metricFilter
	Name  string
	Value float64
}

func (m *MetricSampleValueChecker) String() string {
	return fmt.Sprintf("MetricValueChecker{Labels: %v, Value: %f}", m.Labels, m.Value)
}

func (m *MetricSampleValueChecker) Check(metricFamilies map[string]*dto.MetricFamily) (bool, string) {
	for _, metricFamily := range metricFamilies {
		if metricFamily.GetName() != m.Name {
			continue
		}

		for _, metric := range metricFamily.GetMetric() {
			if !m.isMetricMatch(metric) {
				continue
			}

			var value float64
			if metric.GetGauge() != nil {
				value = metric.GetGauge().GetValue()
			} else if metric.GetCounter() != nil {
				value = metric.GetCounter().GetValue()
			} else if metric.GetSummary() != nil {
				value = metric.GetSummary().GetSampleSum()
			} else if metric.GetHistogram() != nil {
				value = metric.GetHistogram().GetSampleSum()
			} else if metric.GetUntyped() != nil {
				value = metric.GetUntyped().GetValue()
			} else {
				return false, fmt.Sprintf("expected value %f, but got nil in metric %s", value, m.Name)
			}

			if value == m.Value {
				return true, ""
			}
		}
	}
	return false, fmt.Sprintf("expected value %f not found in metric %s", m.Value, m.Name)
}

func NewMetricSampleValueChecker(name string, labels map[string]string, value float64) *MetricSampleValueChecker {
	return &MetricSampleValueChecker{
		metricFilter: newMetricFilter(labels),
		Name:         name,
		Value:        value,
	}
}
