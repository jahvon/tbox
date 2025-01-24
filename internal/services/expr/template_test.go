package expr_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jahvon/flow/internal/services/expr"
)

var _ = Describe("Template", func() {
	var (
		tmpl *expr.Template
		data map[string]interface{}
	)

	BeforeEach(func() {
		data = map[string]interface{}{
			"os":         "linux",
			"arch":       "amd64",
			"store":      map[string]interface{}{"key1": "value1", "key2": 2},
			"ctx":        map[string]interface{}{"workspace": "test_workspace", "namespace": "test_namespace"},
			"workspaces": []string{"test_workspace", "other_workspace"},
			"executables": []map[string]interface{}{
				{"name": "exec1", "tags": []string{"tag"}, "type": "serial"},
				{"name": "exec2", "tags": []string{}, "type": "exec"},
				{"name": "exec3", "tags": []string{"tag", "tag2"}, "type": "exec"},
			},
			"featureEnabled": true,
		}
		tmpl = expr.NewTemplate("test", data)
	})

	Describe("expr evaluation", func() {
		It("evaluates simple expressions", func() {
			err := tmpl.Parse("{{ ctx.workspace }}")
			Expect(err).NotTo(HaveOccurred())

			result, err := tmpl.ExecuteToString()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("test_workspace"))
		})

		It("evaluates boolean expressions", func() {
			err := tmpl.Parse("{{ os == \"linux\" && arch == \"amd64\" }}")
			Expect(err).NotTo(HaveOccurred())

			result, err := tmpl.ExecuteToString()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("true"))
		})

		It("evaluates arithmetic expressions", func() {
			err := tmpl.Parse("{{ store[\"key2\"] * 2 }}")
			Expect(err).NotTo(HaveOccurred())

			result, err := tmpl.ExecuteToString()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("4"))
		})
	})

	Describe("control structures", func() {
		It("handles if/else with expr conditions", func() {
			template := `
				{{- if featureEnabled && ctx.workspace == "test_workspace" }}
				Matched
				{{- else }}
				Unmatched
				{{- end }}
			`
			err := tmpl.Parse(template)
			Expect(err).NotTo(HaveOccurred())

			result, err := tmpl.ExecuteToString()
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(result)).To(Equal("Matched"))
		})

		It("handles range with expr", func() {
			template := `
{{- range filter(executables, {.type == "exec"}) }}
{{ .name }}: {{ .tags }}
{{- end }}
			`
			err := tmpl.Parse(template)
			Expect(err).NotTo(HaveOccurred())

			result, err := tmpl.ExecuteToString()
			Expect(err).NotTo(HaveOccurred())
			expected := "exec2: []\nexec3: [tag tag2]"
			Expect(strings.TrimSpace(result)).To(Equal(expected))
		})

		It("handles with using expr", func() {
			template := `
{{- with ctx }}
Workspace: {{ .workspace }}
Namespace: {{ .namespace }}
{{- end }}
			`
			err := tmpl.Parse(template)
			Expect(err).NotTo(HaveOccurred())

			result, err := tmpl.ExecuteToString()
			Expect(err).NotTo(HaveOccurred())
			expected := "Workspace: test_workspace\nNamespace: test_namespace"
			Expect(strings.TrimSpace(result)).To(Equal(expected))
		})

		It("handles nested control structures with expr", func() {
			GinkgoT().Skip("nested control structures not supported yet")
			template := `
{{- range executables }}
{{- $exec := . }}
{{- if len($exec.tags) > 0 }}
{{ .name }}: {{ .type }}
{{- end }}
{{- end }}
			`
			err := tmpl.Parse(template)
			Expect(err).NotTo(HaveOccurred())

			result, err := tmpl.ExecuteToString()
			Expect(err).NotTo(HaveOccurred())
			expected := "Item 1: 12.089 (with tax)\nItem 3: 16.5 (with tax)"
			Expect(strings.TrimSpace(result)).To(Equal(expected))
		})
	})

	Describe("error handling", func() {
		It("handles invalid expressions", func() {
			err := tmpl.Parse("{{ unknown.field }}")
			Expect(err).NotTo(HaveOccurred())

			_, err = tmpl.ExecuteToString()
			Expect(err).To(HaveOccurred())
		})

		It("handles invalid syntax in if conditions", func() {
			err := tmpl.Parse("{{ if 1 ++ \"2\" }}invalid{{end}}")
			Expect(err).NotTo(HaveOccurred())

			_, err = tmpl.ExecuteToString()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Template with trim markers", func() {
		It("handles trim markers in range", func() {
			template := `start
{{- range workspaces }}
{{ . }}
{{- end }}
end`
			err := tmpl.Parse(template)
			Expect(err).NotTo(HaveOccurred())

			result, err := tmpl.ExecuteToString()
			Expect(err).NotTo(HaveOccurred())
			expected := "start\ntest_workspace\nother_workspace\nend"
			Expect(result).To(Equal(expected))
		})

		It("handles trim markers in if/else", func() {
			template := `start
{{- if featureEnabled }}
enabled
{{- else }}
disabled
{{- end }}
end`
			err := tmpl.Parse(template)
			Expect(err).NotTo(HaveOccurred())

			result, err := tmpl.ExecuteToString()
			Expect(err).NotTo(HaveOccurred())
			expected := "start\nenabled\nend"
			Expect(result).To(Equal(expected))
		})
	})
})
