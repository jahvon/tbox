package tests_test

import (
	stdCtx "context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jahvon/flow/internal/context"
	"github.com/jahvon/flow/tests/utils"
	"github.com/jahvon/flow/types/executable"
)

var _ = Describe("flowfile template commands e2e", Ordered, func() {
	var (
		ctx              *context.Context
		run              *utils.CommandRunner
		template         *executable.Template
		expectedFlowFile *executable.FlowFile
	)

	BeforeAll(func() {
		ctx = utils.NewContext(stdCtx.Background(), GinkgoT())
		run = utils.NewE2ECommandRunner()
		workDir, err := os.MkdirTemp("", "flowfile-template-e2e")
		Expect(err).NotTo(HaveOccurred())
		tmpl := executable.FlowFile{
			Namespace:   "test",
			Description: "Template test flowfile",
			Tags:        []string{"test"},
			Executables: []*executable.Executable{
				{
					Verb: "exec",
					Name: "{{ name }}",
					Exec: &executable.ExecExecutableType{Cmd: fmt.Sprintf("echo '%s'", "{{ form['Msg'] }}")}},
			},
		}
		tmplStr, err := tmpl.YAML()
		Expect(err).NotTo(HaveOccurred())
		template = &executable.Template{
			Template: tmplStr,
			Form: executable.FormFields{
				&executable.Field{
					Key:     "Name",
					Prompt:  "Enter a name",
					Default: "test",
				},
				&executable.Field{
					Key:     "Msg",
					Prompt:  "Enter a message",
					Default: "Hello, world!",
				},
			},
			Artifacts: []executable.Artifact{
				{SrcName: "artifact1"},
				{SrcName: "artifact2", DstName: "artifact2-renamed"},
			},
			PreRun: []executable.TemplateRefConfig{
				{Ref: "exec examples:simple-print", Args: []string{"test"}},
			},
			PostRun: []executable.TemplateRefConfig{
				{
					Cmd: "touch {{ name }}",
				},
			},
		}
		template.SetContext("e2e", filepath.Join(workDir, "flowfile.tmpl.flow"))
		data, err := template.YAML()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(workDir, "flowfile.tmpl.flow"), []byte(data), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workDir, "artifact1"), []byte("artifact1"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workDir, "artifact2"), []byte("artifact2"), 0644)).To(Succeed())

		expectedFlowFile = &executable.FlowFile{
			Namespace:   "test",
			Description: "Template test flowfile",
			Tags:        []string{"test"},
			Executables: []*executable.Executable{
				{
					Verb: "exec",
					Name: "test",
					Exec: &executable.ExecExecutableType{Cmd: "echo 'Hello, world!'"}},
			},
		}
		expectedFlowFile.SetContext(
			ctx.CurrentWorkspace.AssignedName(),
			ctx.CurrentWorkspace.Location(),
			filepath.Join(workDir, "flowfile.flow"),
		)
	})

	BeforeEach(func() {
		utils.ResetTestContext(ctx, GinkgoT())
	})

	AfterEach(func() {
		ctx.Finalize()
	})

	When("registering a new template (flow template register)", func() {
		It("should complete successfully", func() {
			stdOut := ctx.StdOut()
			err := run.Run(ctx, "template", "register", "--verbosity", "-1", template.Name(), template.Location())
			Expect(err).ToNot(HaveOccurred())
			out, err := readFileContent(stdOut)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring(fmt.Sprintf("Template %s set", template.Name())))
		})
	})

	When("getting a registered template (flow template view)", func() {
		It("should return the template", func() {
			stdOut := ctx.StdOut()
			err := run.Run(ctx, "template", "view", "-t", template.Name(), "-o", "yaml")
			Expect(err).ToNot(HaveOccurred())
			out, err := readFileContent(stdOut)
			Expect(err).NotTo(HaveOccurred())
			str, err := template.YAML()
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring(str))
		})
	})

	When("getting a template by path (flow template view)", func() {
		It("should return the template", func() {
			stdOut := ctx.StdOut()
			err := run.Run(ctx, "template", "view", "-f", template.Location(), "-o", "yaml")
			Expect(err).ToNot(HaveOccurred())
			out, err := readFileContent(stdOut)
			Expect(err).NotTo(HaveOccurred())
			str, err := template.YAML()
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring(str))
		})
	})

	When("Listing all registered templates (flow template list)", func() {
		It("should return the list of templates", func() {
			stdOut := ctx.StdOut()
			Expect(run.Run(ctx, "template", "list", "-o", "yaml")).To(Succeed())
			out, err := readFileContent(stdOut)
			Expect(err).NotTo(HaveOccurred())
			// tabs may be present so instead of checking for exact match, we check for length
			str, err := template.YAML()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(out)).To(BeNumerically(">", len(str)))
		})
	})

	When("Rendering a template (flow template generate)", func() {
		It("should process the template options and render the flowfile", func() {
			name := "test"
			outputDir := filepath.Join(ctx.CurrentWorkspace.Location(), "output")
			reader, writer, err := os.Pipe()
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Write([]byte("test\nhello\n"))
			Expect(err).ToNot(HaveOccurred())

			ctx.SetIO(reader, ctx.StdOut())
			Expect(run.Run(ctx, "template", "generate", name, "-t", template.Name(), "-o", outputDir)).To(Succeed())
			out, err := readFileContent(ctx.StdOut())
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring(fmt.Sprintf("Template '%s' rendered successfully", name)))
		})
	})
})
