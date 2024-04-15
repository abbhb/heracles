package core

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/mrlyc/heracles/log"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
)

var ErrCheck = errors.New("check failed")

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

func (r *Runner) FetchMetricFamilies(ctx context.Context, baseUrl string) (map[string]*dto.MetricFamily, error) {
	url, err := url.JoinPath(baseUrl, r.config.GetString("exporter.path"))
	if err != nil {
		return nil, eris.Wrap(err, "failed to join url")
	}

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, eris.Wrap(err, "failed to fetch metrics")
	}

	log.Infof("fetch metrics from %s, status code: %d", url, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, eris.New("failed to fetch metrics: " + resp.Status)
	}

	defer resp.Body.Close()

	var parser expfmt.TextParser
	metricFamily, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, eris.Wrap(err, "failed to parse metrics")
	}

	log.Infof("found %d metric families", len(metricFamily))

	return metricFamily, nil
}

func (r *Runner) Run(ctx context.Context, callback func(ctx context.Context, metricFamilies map[string]*dto.MetricFamily) error) error {
	if err := r.SetupFixtures(ctx); err != nil {
		return eris.Wrap(err, "failed to setup fixtures")
	}

	defer func() {
		if err := r.TearDownFixtures(ctx); err != nil {
			log.Errorf("failed to tear down fixtures: %+v", err)
		}
	}()

	baseUrl, err := r.exporter.Start(ctx)
	if err != nil {
		return eris.Wrap(err, "failed to start exporter")
	}

	metricFamilies, err := r.FetchMetricFamilies(ctx, baseUrl)
	if err != nil {
		return eris.Wrap(err, "failed to fetch metrics")
	}

	err = callback(ctx, metricFamilies)
	if err != nil {
		return eris.Wrap(err, "callback failed")
	}

	return nil
}

func NewRunner(exporter Exporter, fixtures []Fixture, conf *viper.Viper) *Runner {
	return &Runner{
		exporter:   exporter,
		fixtures:   fixtures,
		config:     conf,
		httpClient: http.DefaultClient,
	}
}

type MetricChecker struct {
	*Runner
}

func (c *MetricChecker) CheckMetrics(ctx context.Context, metricFamily map[string]*dto.MetricFamily) error {
	checkers, err := c.BuildChecker()
	if err != nil {
		return eris.Wrap(err, "failed to build checkers")
	}

	var messages []string
	for _, checker := range checkers {
		ok, message := checker.Check(metricFamily)
		if !ok {
			messages = append(messages, message)
		}
	}

	if len(messages) > 0 {
		return eris.Wrap(ErrCheck, "details: \n"+strings.Join(messages, "\n"))
	}

	return nil
}

func (c *MetricChecker) BuildChecker() ([]MetricFamiliesChecker, error) {
	checkerBuilder := NewMetricFamiliesCheckerBuilder()

	disallowedMetrics := c.config.GetStringSlice("exporter.disallowed_metrics")
	if len(disallowedMetrics) != 0 {
		checkerBuilder.DisallowedMetrics(disallowedMetrics)
	}

	if !c.config.GetBool("exporter.allow_empty") {
		checkerBuilder.EmptyMetricsChecker()
	}

	var metrics []metricsConfig
	err := c.config.UnmarshalKey("exporter.metrics", &metrics)
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

func (c *MetricChecker) Check(ctx context.Context) error {
	return c.Run(ctx, c.CheckMetrics)
}

func NewMetricChecker(exporter Exporter, fixtures []Fixture, conf *viper.Viper) *MetricChecker {
	return &MetricChecker{
		Runner: NewRunner(exporter, fixtures, conf),
	}
}
