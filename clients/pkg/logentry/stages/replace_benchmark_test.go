package stages

import (
	"regexp"
	"testing"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	util_log "github.com/grafana/loki/v3/pkg/util/log"
)

// 基准测试数据
var benchmarkTestCases = []struct {
	name        string
	config      string
	entry       string
	description string
}{
	{
		name: "SimpleReplace",
		config: `
pipeline_stages:
- replace:
    expression: "11.11.11.11 - (\\S+) .*"
    replace: "dummy"
`,
		entry:       `11.11.11.11 - frank [25/Jan/2000:14:00:01 -0500] "GET /1986.js HTTP/1.1" 200 932 "-" "Mozilla/5.0"`,
		description: "简单替换测试",
	},
	{
		name: "ComplexRegex",
		config: `
pipeline_stages:
- replace:
    expression: "^(?P<ip>\\S+) (?P<identd>\\S+) (?P<user>\\S+) \\[(?P<timestamp>[\\w:/]+\\s[+\\-]\\d{4})\\] \"(?P<action>\\S+)\\s?(?P<path>\\S+)?\\s?(?P<protocol>\\S+)?\" (?P<status>\\d{3}|-) (\\d+|-)\\s?\"?(?P<referer>[^\"]*)\"?\\s?\"?(?P<useragent>[^\"]*)?\"?$"
    replace: '{{ if eq .Value "200" }}{{ Replace .Value "200" "HttpStatusOk" -1 }}{{ else }}{{ .Value | ToUpper }}{{ end }}'
`,
		entry:       `11.11.11.11 - frank [25/Jan/2000:14:00:01 -0500] "GET /1986.js HTTP/1.1" 200 932 "-" "Mozilla/5.0"`,
		description: "复杂正则表达式测试",
	},
	{
		name: "MultipleMatches",
		config: `
pipeline_stages:
- replace:
    expression: "(\\d{2}\\.\\d{2}\\.\\d{2}\\.\\d{2}) - (\\S+)"
    replace: "{{ .Value | ToUpper }}"
`,
		entry:       `11.11.11.11 - frank 12.12.12.12 - john 13.13.13.13 - mary`,
		description: "多次匹配测试",
	},
	{
		name: "TemplateWithSource",
		config: `
pipeline_stages:
- json:
    expressions:
      level:
      msg:
- replace:
    expression: "\\S+ - \"POST (\\S+) .*"
    source: msg
    replace: "/loki/api/v1/push/"
`,
		entry:       `{"time":"2019-01-01T01:00:00.000000001Z", "level": "info", "msg": "11.11.11.11 - \"POST /loki/api/v1/push/ HTTP/1.1\" 200 932"}`,
		description: "带源字段的模板测试",
	},
}

// BenchmarkReplaceStage_Process 基准测试主函数
func BenchmarkReplaceStage_Process(b *testing.B) {
	for _, tc := range benchmarkTestCases {
		b.Run(tc.name, func(b *testing.B) {
			// 创建pipeline
			pl, err := NewPipeline(util_log.Logger, loadConfig(tc.config), nil, prometheus.DefaultRegisterer)
			if err != nil {
				b.Fatal(err)
			}

			// 预热
			for i := 0; i < 100; i++ {
				processEntries(pl, newEntry(nil, nil, tc.entry, time.Now()))
			}

			// 重置计时器
			b.ResetTimer()

			// 基准测试
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					processEntries(pl, newEntry(nil, nil, tc.entry, time.Now()))
				}
			})
		})
	}
}

// BenchmarkReplaceStage_Memory 内存分配基准测试
func BenchmarkReplaceStage_Memory(b *testing.B) {
	for _, tc := range benchmarkTestCases {
		b.Run(tc.name, func(b *testing.B) {
			pl, err := NewPipeline(util_log.Logger, loadConfig(tc.config), nil, prometheus.DefaultRegisterer)
			if err != nil {
				b.Fatal(err)
			}

			// 预热
			for i := 0; i < 100; i++ {
				processEntries(pl, newEntry(nil, nil, tc.entry, time.Now()))
			}

			// 重置计时器
			b.ResetTimer()

			// 基准测试
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					processEntries(pl, newEntry(nil, nil, tc.entry, time.Now()))
				}
			})
		})
	}
}

// BenchmarkReplaceStage_Concurrent 并发性能测试
func BenchmarkReplaceStage_Concurrent(b *testing.B) {
	for _, tc := range benchmarkTestCases {
		b.Run(tc.name, func(b *testing.B) {
			pl, err := NewPipeline(util_log.Logger, loadConfig(tc.config), nil, prometheus.DefaultRegisterer)
			if err != nil {
				b.Fatal(err)
			}

			// 预热
			for i := 0; i < 100; i++ {
				processEntries(pl, newEntry(nil, nil, tc.entry, time.Now()))
			}

			// 重置计时器
			b.ResetTimer()

			// 并发基准测试
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					processEntries(pl, newEntry(nil, nil, tc.entry, time.Now()))
				}
			})
		})
	}
}

// BenchmarkReplaceStage_TemplateParsing 模板解析性能测试
func BenchmarkReplaceStage_TemplateParsing(b *testing.B) {
	templateStr := `{{ if eq .Value "200" }}{{ Replace .Value "200" "HttpStatusOk" -1 }}{{ else }}{{ .Value | ToUpper }}{{ end }}`
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		template.New("pipeline_template").Funcs(functionMap).Parse(templateStr)
	}
}

// BenchmarkReplaceStage_RegexCompilation 正则表达式编译性能测试
func BenchmarkReplaceStage_RegexCompilation(b *testing.B) {
	regexStr := `^(?P<ip>\S+) (?P<identd>\S+) (?P<user>\S+) \[(?P<timestamp>[\w:/]+\s[+\-]\d{4})\] "(?P<action>\S+)\s?(?P<path>\S+)?\s?(?P<protocol>\S+)?" (?P<status>\d{3}|-) (\d+|-)\s?"?(?P<referer>[^"]*)"?\s?"?(?P<useragent>[^"]*)?"?$`
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		regexp.Compile(regexStr)
	}
} 