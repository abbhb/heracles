package core

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/mrlyc/heracles/log"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rotisserie/eris"
	"gopkg.in/yaml.v3"
)

var ErrCheck = errors.New("check failed")

type CheckReport struct {
	Success bool                         `json:"success"`
	Metrics map[string]*dto.MetricFamily `json:"inputs"`
	Results map[string]string            `json:"outputs"`
}

func (c *CheckReport) Yaml() ([]byte, error) {
	return yaml.Marshal(c)
}

type MetricSample struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  *float64          `json:"value"`
}

type MetricsConfig struct {
	Name             string         `mapstructure:"name"`
	Type             string         `mapstructure:"type"`
	Labels           []string       `mapstructure:"labels"`
	DisallowedLabels []string       `mapstructure:"disallowed_labels"`
	Samples          []MetricSample `mapstructure:"samples"`
}

type Runner struct {
	exporter     Exporter
	fixtures     []Fixture
	httpClient   HTTPClient
	metricPath   string
	waitDuration time.Duration
}

func (r *Runner) SetupFixtures(ctx context.Context) ([]Fixture, error) {
	setups := make([]Fixture, 0, len(r.fixtures))
	for _, fixture := range r.fixtures {
		log.Debugf("setting up fixture: %s", fixture)
		if err := fixture.Setup(ctx); err != nil {
			return setups, eris.Wrap(err, "failed to setup fixture")
		}
		setups = append(setups, fixture)
	}

	return setups, nil
}

func (r *Runner) TearDownFixtures(ctx context.Context, fixtures []Fixture) (ok bool) {
	ok = true

	for i := len(fixtures) - 1; i >= 0; i-- {
		log.Debugf("tearing down fixture: %s", r.fixtures[i])
		err := fixtures[i].TearDown(ctx)
		if err != nil {
			log.Errorf("failed to tear down fixtures: %+v", err)
			ok = false
		}
	}

	return ok
}

func (r *Runner) FetchMetricFamilies(ctx context.Context, baseUrl string) (map[string]*dto.MetricFamily, error) {
	url, err := url.JoinPath(baseUrl, r.metricPath)
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

	log.Infof("found %d metrics", len(metricFamily))

	return metricFamily, nil
}

func (r *Runner) Run(ctx context.Context, callback func(ctx context.Context, metricFamilies map[string]*dto.MetricFamily) error) error {
	fixtures, err := r.SetupFixtures(ctx)
	defer func() {
		r.TearDownFixtures(ctx, fixtures)
	}()

	if err != nil {
		return err
	}

	baseUrl, err := r.exporter.Start(ctx)
	if err != nil {
		return eris.Wrap(err, "failed to start exporter")
	}

	log.Infof("waiting for %s", r.waitDuration)
	time.Sleep(r.waitDuration)

	metricFamilies, err := r.FetchMetricFamilies(ctx, baseUrl)
	if err != nil {
		return eris.Wrap(err, "failed to fetch metrics")
	}

	err = callback(ctx, metricFamilies)
	if err != nil {
		return err
	}

	return nil
}

func NewRunner(exporter Exporter, fixtures []Fixture, metricPath string, waitDuration time.Duration) *Runner {
	return &Runner{
		exporter:     exporter,
		fixtures:     fixtures,
		httpClient:   http.DefaultClient,
		metricPath:   metricPath,
		waitDuration: waitDuration,
	}
}

type MetricChecker struct {
	*Runner
	disallowedMetrics []string
	allowEmpty        bool
	metrics           []MetricsConfig
}

func (c *MetricChecker) CheckMetrics(ctx context.Context, metricFamily map[string]*dto.MetricFamily) (*CheckReport, error) {
	checkers, err := c.BuildChecker()
	if err != nil {
		return nil, eris.Wrap(err, "failed to build checkers")
	}

	var returnedError error
	report := &CheckReport{
		Success: true,
		Metrics: metricFamily,
		Results: make(map[string]string, len(checkers)),
	}

	for _, checker := range checkers {
		log.Debugf("checking metrics by checker %v", checker)
		ok, message := checker.Check(metricFamily)
		if !ok {
			log.Errorf("metrics check failed, %v", message)
			returnedError = ErrCheck
		}

		report.Results[checker.String()] = message
	}

	return report, returnedError
}

func (c *MetricChecker) BuildChecker() ([]MetricFamiliesChecker, error) {
	checkerBuilder := NewMetricFamiliesCheckerBuilder()

	disallowedMetrics := c.disallowedMetrics
	if len(disallowedMetrics) != 0 {
		checkerBuilder.DisallowedMetrics(disallowedMetrics)
	}

	if !c.allowEmpty {
		checkerBuilder.EmptyMetricsChecker()
	}

	for _, metric := range c.metrics {
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

		for _, sample := range metric.Samples {
			checkerBuilder.MetricSampleChecker(metric.Name, sample.Labels)

			if sample.Value != nil {
				checkerBuilder.MetricSampleValueChecker(metric.Name, sample.Labels, *sample.Value)
			}
		}
	}

	return checkerBuilder.Build(), nil
}

func (c *MetricChecker) Check(ctx context.Context) (checkReport *CheckReport, checkErr error) {
	checkErr = c.Run(ctx, func(ctx context.Context, metricFamilies map[string]*dto.MetricFamily) error {
		report, err := c.CheckMetrics(ctx, metricFamilies)
		checkReport = report
		return err
	})
	return
}

func NewMetricChecker(
	exporter Exporter,
	fixtures []Fixture,
	metricPath string,
	disallowedMetrics []string,
	allowEmpty bool,
	metrics []MetricsConfig,
	waitDuration time.Duration,
) *MetricChecker {
	return &MetricChecker{
		Runner:            NewRunner(exporter, fixtures, metricPath, waitDuration),
		disallowedMetrics: disallowedMetrics,
		allowEmpty:        allowEmpty,
		metrics:           metrics,
	}
}
