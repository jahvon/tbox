package rest_test

import (
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jahvon/flow/internal/services/rest"
)

func TestRest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rest Suite")
}

var _ = Describe("Rest", func() {
	Context("SendRequest", func() {
		It("should return error when invalid URL is provided", func() {
			req := &rest.Request{
				URL:     "invalid_url",
				Method:  "GET",
				Timeout: 30 * time.Second,
			}
			_, err := rest.SendRequest(req, []int{http.StatusOK})
			Expect(err).To(HaveOccurred())
		})

		It("should return error when unexpected status code is received", func() {
			req := &rest.Request{
				URL:     "https://httpbin.org/status/500",
				Method:  "GET",
				Timeout: 30 * time.Second,
			}
			_, err := rest.SendRequest(req, []int{http.StatusOK})
			Expect(err).To(Equal(rest.ErrUnexpectedStatusCode))
		})

		It("should return the correct body when a valid request is made", func() {
			req := &rest.Request{
				URL:     "https://httpbin.org/get",
				Method:  "GET",
				Timeout: 30 * time.Second,
			}
			resp, err := rest.SendRequest(req, []int{http.StatusOK})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Body).To(ContainSubstring("\"url\": \"https://httpbin.org/get\""))
		})

		It("should timeout when the request takes longer than the specified timeout", func() {
			req := &rest.Request{
				URL:     "https://httpbin.org/delay/3",
				Method:  "GET",
				Timeout: 1 * time.Second,
			}
			_, err := rest.SendRequest(req, []int{http.StatusOK})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Client.Timeout exceeded while awaiting headers"))
		})

		It("should return the correct headers when a valid request is made", func() {
			req := &rest.Request{
				URL:     "https://httpbin.org/headers",
				Method:  "GET",
				Headers: map[string]string{"Test-Header": "Test-Value"},
				Timeout: 30 * time.Second,
			}
			resp, err := rest.SendRequest(req, []int{http.StatusOK})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Body).To(ContainSubstring("\"Test-Header\": \"Test-Value\""))
		})
	})
})
