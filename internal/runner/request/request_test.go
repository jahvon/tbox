package request_test

import (
	stdCtx "context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/jahvon/flow/internal/runner"
	"github.com/jahvon/flow/internal/runner/engine/mocks"
	"github.com/jahvon/flow/internal/runner/request"
	testUtils "github.com/jahvon/flow/tests/utils"
	"github.com/jahvon/flow/types/executable"
)

func TestRequest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Request Suite")
}

var _ = Describe("Request Runner", func() {
	var (
		requestRnr runner.Runner
		ctx        *testUtils.ContextWithMocks
		mockEngine *mocks.MockEngine
	)

	BeforeEach(func() {
		ctx = testUtils.NewContextWithMocks(stdCtx.Background(), GinkgoT())
		requestRnr = request.NewRunner()
		ctrl := gomock.NewController(GinkgoT())
		mockEngine = mocks.NewMockEngine(ctrl)
	})

	Context("Name", func() {
		It("should return the correct requestRnr name", func() {
			Expect(requestRnr.Name()).To(Equal("request"))
		})
	})

	Context("IsCompatible", func() {
		It("should return false when executable is nil", func() {
			Expect(requestRnr.IsCompatible(nil)).To(BeFalse())
		})

		It("should return false when executable type is nil", func() {
			executable := &executable.Executable{}
			Expect(requestRnr.IsCompatible(executable)).To(BeFalse())
		})

		It("should return true when executable type is serial", func() {
			executable := &executable.Executable{
				Request: &executable.RequestExecutableType{},
			}
			Expect(requestRnr.IsCompatible(executable)).To(BeTrue())
		})
	})

	Describe("Exec", func() {
		It("should send a GET request and log the response", func() {
			exec := &executable.Executable{
				Request: &executable.RequestExecutableType{
					URL:         "https://httpbin.org/get",
					Method:      executable.RequestExecutableTypeMethodGET,
					LogResponse: true,
				},
			}

			ctx.Logger.EXPECT().Infox(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
			err := requestRnr.Exec(ctx.Ctx, exec, mockEngine, make(map[string]string))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should send a POST request with a body and log the response", func() {
			exec := &executable.Executable{
				Request: &executable.RequestExecutableType{
					URL:         "https://httpbin.org/post",
					Method:      executable.RequestExecutableTypeMethodPOST,
					Body:        `{"key": "value"}`,
					LogResponse: true,
				},
			}

			ctx.Logger.EXPECT().Infox(gomock.Any(), gomock.Any(), gomock.Regex("value")).Times(1)
			err := requestRnr.Exec(ctx.Ctx, exec, mockEngine, make(map[string]string))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should save the response to a file", func() {
			exec := &executable.Executable{
				Request: &executable.RequestExecutableType{
					URL:    "https://httpbin.org/get",
					Method: executable.RequestExecutableTypeMethodGET,
					ResponseFile: &executable.RequestResponseFile{
						Filename: "response.json",
						Dir:      executable.Directory("//"),
						SaveAs:   executable.RequestResponseFileSaveAsJson,
					},
				},
			}
			exec.SetContext(ctx.Ctx.CurrentWorkspace.AssignedName(), ctx.Ctx.CurrentWorkspace.Location(), "", "")

			ctx.Logger.EXPECT().Infof(gomock.Any(), gomock.Any()).Times(2)
			err := requestRnr.Exec(ctx.Ctx, exec, mockEngine, make(map[string]string))
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Clean(filepath.Join(ctx.Ctx.CurrentWorkspace.Location(), "response.json")))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should transform the response when specified", func() {
			exec := &executable.Executable{
				Request: &executable.RequestExecutableType{
					URL:               "https://httpbin.org/get",
					Method:            executable.RequestExecutableTypeMethodGET,
					TransformResponse: `upper(body)`,
					LogResponse:       true,
				},
			}

			ctx.Logger.EXPECT().Infox(gomock.Any(), gomock.Any(), gomock.Regex("HTTPS://HTTPBIN.ORG")).Times(1)
			err := requestRnr.Exec(ctx.Ctx, exec, mockEngine, make(map[string]string))
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
