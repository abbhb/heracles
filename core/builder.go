package core

type MetricFamiliesCheckerBuilder struct {
	globalCheckers  []MetricFamiliesChecker
	metricsCheckers map[string][]MetricFamiliesChecker
}

// GlobalCheckers 往全局检查器列表中添加一个 MetricFamiliesChecker。
func (b *MetricFamiliesCheckerBuilder) GlobalCheckers(checker MetricFamiliesChecker) {
	b.globalCheckers = append(b.globalCheckers, checker)
}

// MetricsCheckers 往指定指标的检查器列表中添加一个检查器。
func (b *MetricFamiliesCheckerBuilder) MetricsCheckers(metric string, checkers ...MetricFamiliesChecker) {
	metricCheckers, ok := b.metricsCheckers[metric]
	if !ok {
		b.metricsCheckers[metric] = checkers
	} else {
		b.metricsCheckers[metric] = append(metricCheckers, checkers...)
	}
}

// DisallowedMetrics 添加一个禁止指定指标的检查器。
func (b *MetricFamiliesCheckerBuilder) DisallowedMetrics(metrics []string) {
	b.GlobalCheckers(NewDisallowCertainMetricsChecker(metrics))
}

// EmptyMetricsChecker 添加一个禁止空指标的检查器。
func (b *MetricFamiliesCheckerBuilder) EmptyMetricsChecker() {
	b.GlobalCheckers(NewDisallowEmptyMetricsChecker())
}

// MetricExistsChecker 添加一个确保指定指标存在的检查器。
func (b *MetricFamiliesCheckerBuilder) MetricExistsChecker(metric string) {
	b.MetricsCheckers(metric, NewSingleMetricExistsChecker(metric))
}

// MetricTypeChecker 添加一个确保指定指标类型正确的检查器。
func (b *MetricFamiliesCheckerBuilder) MetricTypeChecker(metric string, metricType string) {
	b.MetricsCheckers(metric, NewSingleMetricTypeChecker(metric, metricType))
}

// MetricLabelChecker 添加一个确保指定指标有正确标签的检查器。
func (b *MetricFamiliesCheckerBuilder) MetricLabelChecker(metric string, label ...string) {
	b.MetricsCheckers(metric, NewMetricLabelChecker(metric, label))
}

// MetricLabelDisallowChecker 添加一个禁止指定指标的指定标签的检查器。
func (b *MetricFamiliesCheckerBuilder) MetricLabelDisallowChecker(metric string, label ...string) {
	b.MetricsCheckers(metric, NewMetricLabelDisallowChecker(metric, label))
}

// Build 将所有检查器组合成一个切片并返回。
func (b *MetricFamiliesCheckerBuilder) Build() []MetricFamiliesChecker {
	checkers := b.globalCheckers
	for _, metricCheckers := range b.metricsCheckers {
		checkers = append(checkers, metricCheckers...)
	}
	return checkers
}

func NewMetricFamiliesCheckerBuilder() *MetricFamiliesCheckerBuilder {
	return &MetricFamiliesCheckerBuilder{
		globalCheckers:  make([]MetricFamiliesChecker, 0),
		metricsCheckers: make(map[string][]MetricFamiliesChecker),
	}
}

type FixtureBuilder struct {
	fixtures []Fixture
}

func (b *FixtureBuilder) Build() []Fixture {
	return b.fixtures
}

func (b *FixtureBuilder) AppendScriptFixtures(fixtures ...ScriptFixture) {
	for _, fixture := range fixtures {
		b.fixtures = append(b.fixtures, fixture)
	}
}

func NewFixtureBuilder(fixture ...Fixture) *FixtureBuilder {
	return &FixtureBuilder{
		fixtures: fixture,
	}
}
