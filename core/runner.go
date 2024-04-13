package core

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/prometheus/common/expfmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
)

type metricsConfig struct {
	Name             string   `mapstructure:"name"`
	Type             string   `mapstructure:"type"`
	Labels           []string `mapstructure:"labels"`
	DisallowedLabels []string `mapstructure:"disallowed_labels"`
}

type Runner struct {
	exporter   Exporter
	fixtures   []Fixture
	config     *viper.Viper
	httpClient HTTPClient
}

func (r *Runner) SetupFixtures(ctx context.Context) error {
	for _, fixture := range r.fixtures {
		if err := fixture.Setup(ctx); err != nil {
			return eris.Wrap(err, "failed to setup fixture")
		}
	}

	return nil
}

func (r *Runner) TearDownFixtures(ctx context.Context) error {
	for _, fixture := range r.fixtures {
		if err := fixture.TearDown(ctx); err != nil {
			return eris.Wrap(err, "failed to tear down fixture")
		}
	}

	return nil
}

func (r *Runner) FetchMetrics(ctx context.Context, baseUrl string) (io.ReadCloser, error) {
	url, err := url.JoinPath(baseUrl, r.config.GetString("exporter.path"))
	if err != nil {
		return nil, eris.Wrap(err, "failed to join url")
	}

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, eris.Wrap(err, "failed to fetch metrics")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, eris.New("failed to fetch metrics: " + resp.Status)
	}

	return resp.Body, nil
}

func (r *Runner) CheckMetrics(reader io.Reader, checkers []MetricFamiliesChecker) error {
	var parser expfmt.TextParser
	metricFamily, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return eris.Wrap(err, "failed to parse metrics")
	}

	var messages []string
	for _, checker := range checkers {
		ok, message := checker.Check(metricFamily)
		if !ok {
			messages = append(messages, message)
		}
	}

	if len(messages) > 0 {
		return eris.New("metrics check failed: \n" + strings.Join(messages, "\n"))
	}

	return nil
}

func (r *Runner) BuildChecker() ([]MetricFamiliesChecker, error) {
	checkerBuilder := NewMetricFamiliesCheckerBuilder()

	disallowedMetrics := r.config.GetStringSlice("exporter.disallowed_metrics")
	if len(disallowedMetrics) != 0 {
		checkerBuilder.DisallowedMetrics(disallowedMetrics)
	}

	if !r.config.GetBool("exporter.allow_empty") {
		checkerBuilder.EmptyMetricsChecker()
	}

	var metrics []metricsConfig
	err := r.config.UnmarshalKey("exporter.metrics", &metrics)
	if err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal metrics")
	}

	for _, metric := range metrics {
		checkerBuilder.MetricExistsChecker(metric.Name)

		if metric.Type != "" {
			checkerBuilder.MetricTypeChecker(metric.Name, metric.Type)
		}

		if len(metric.Labels) != 0 {
			checkerBuilder.MetricLabelChecker(metric.Name, metric.Labels...)
		}

		if len(metric.DisallowedLabels) != 0 {
			checkerBuilder.MetricLabelDisallowChecker(metric.Name, metric.DisallowedLabels...)
		}
	}

	return checkerBuilder.Build(), nil
}

func (r *Runner) Run(ctx context.Context) error {
	if err := r.SetupFixtures(ctx); err != nil {
		return eris.Wrap(err, "failed to setup fixtures")
	}

	baseUrl, err := r.exporter.Start(ctx)
	if err != nil {
		return eris.Wrap(err, "failed to start exporter")
	}

	reader, err := r.FetchMetrics(ctx, baseUrl)
	if err != nil {
		return eris.Wrap(err, "failed to fetch metrics")
	}
	defer reader.Close()

	checkers, err := r.BuildChecker()
	if err != nil {
		return eris.Wrap(err, "failed to build checkers")
	}

	if err := r.CheckMetrics(reader, checkers); err != nil {
		return eris.Wrap(err, "failed to check metrics")
	}

	if err := r.TearDownFixtures(ctx); err != nil {
		return eris.Wrap(err, "failed to tear down fixtures")
	}

	return nil
}

func NewRunner(exporter Exporter, fixtures []Fixture, conf *viper.Viper) *Runner {
	return &Runner{
		exporter: exporter,
		fixtures: fixtures,
		config:   conf,
	}
}
