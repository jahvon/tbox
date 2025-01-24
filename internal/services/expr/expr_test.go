package expr_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jahvon/flow/internal/services/expr"
)

func TestExpr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expr Suite")
}

var _ = Describe("Expr", func() {
	Describe("IsTruthy", func() {
		It("should evaluate truthy expressions correctly", func() {
			tests := []struct {
				expr     string
				env      *expr.ExpressionData
				expected bool
			}{
				{"true", nil, true},
				{"false", nil, false},
				{"1", nil, true},
				{"0", nil, false},
				{`"true"`, nil, true},
				{`"false"`, nil, false},
			}

			for _, test := range tests {
				result, err := expr.IsTruthy(test.expr, test.env)
				Expect(err).NotTo(HaveOccurred())
				By("testing expression: " + test.expr)
				Expect(result).To(Equal(test.expected))
			}
		})
	})

	Describe("Evaluate", func() {
		It("should evaluate expressions correctly", func() {
			tests := []struct {
				expr     string
				env      *expr.ExpressionData
				expected interface{}
			}{
				{"1 + 1", nil, 2},
				{"true && false", nil, false},
				{`"hello" + " " + "world"`, nil, "hello world"},
			}

			for _, test := range tests {
				result, err := expr.Evaluate(test.expr, test.env)
				Expect(err).NotTo(HaveOccurred())
				By("testing expression: " + test.expr)
				Expect(result).To(Equal(test.expected))
			}
		})
	})

	Describe("EvaluateString", func() {
		It("should evaluate string expressions correctly", func() {
			tests := []struct {
				expr     string
				env      *expr.ExpressionData
				expected string
			}{
				{`"hello"`, nil, "hello"},
				{`"foo" + "bar"`, nil, "foobar"},
			}

			for _, test := range tests {
				result, err := expr.EvaluateString(test.expr, test.env)
				Expect(err).NotTo(HaveOccurred())
				By("testing expression: " + test.expr)
				Expect(result).To(Equal(test.expected))
			}
		})
	})

	Describe("ExpressionData", func() {
		var (
			data *expr.ExpressionData
		)

		BeforeEach(func() {
			data = &expr.ExpressionData{
				OS:   "linux",
				Arch: "amd64",
				Ctx: &expr.CtxData{
					Workspace:     "test_workspace",
					Namespace:     "test_namespace",
					WorkspacePath: "/path/to/workspace",
					FlowFilePath:  "/path/to/flowfile",
					FlowFileDir:   "/path/to",
				},
				Store: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Env: map[string]string{
					"ENV_VAR1": "env_value1",
					"ENV_VAR2": "env_value2",
				},
			}
		})

		Describe("Evaluate complex expressions", func() {
			It("should evaluate various expressions correctly", func() {
				tests := []struct {
					expr     string
					expected interface{}
				}{
					{"1 + 1", 2},
					{"true && false", false},
					{`"hello" + " " + "world"`, "hello world"},
					{`store["key1"]`, "value1"},
					{`env["ENV_VAR1"]`, "env_value1"},
					{`os == "linux"`, true},
					{`arch == "amd64"`, true},
					{`ctx.workspace == "test_workspace"`, true},
				}

				for _, test := range tests {
					result, err := expr.Evaluate(test.expr, data)
					Expect(err).NotTo(HaveOccurred())
					By("testing expression: " + test.expr)
					Expect(result).To(Equal(test.expected))
				}
			})
		})
	})
})
