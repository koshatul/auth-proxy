package httpauth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/koshatul/auth-proxy/httpauth"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	cache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const successContent string = "Hello World!"

var _ = Describe("httpauth", func() {

	authFunc := func(username string, password string, r *http.Request) (string, bool) {
		if v, ok := map[string]string{
			"test": "valid-pass",
		}[username]; ok {
			if strings.Compare(v, password) == 0 {
				return username, true
			}
		}

		return "", false
	}

	var (
		logcore zapcore.Core
		logger  *zap.Logger
		ts      *httptest.Server
	)

	BeforeEach(func() {
		// logcore, logobs = observer.New(zap.DebugLevel)
		logcore, _ = observer.New(zap.DebugLevel)
		logger = zap.New(logcore)

		authenticator := &httpauth.BasicAuthHandler{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, successContent)
			}),
			RemoveAuth: true,
			BasicAuthWrapper: &httpauth.BasicAuthWrapper{
				Cache:         cache.New(time.Minute, time.Minute),
				Realm:         "im-a-test-realm",
				AuthFunc:      authFunc,
				Logger:        logger,
				CacheDuration: time.Minute,
			},
		}

		ts = httptest.NewTLSServer(authenticator)
	})

	AfterEach(func() {
		ts.Close()
	})

	expectSuccessBody := func(res *http.Response) {
		p := make([]byte, len(successContent))
		_, err := gbytes.TimeoutReader(res.Body, time.Second).Read(p)
		Expect(err).NotTo(HaveOccurred())
		Expect(p).Should(Equal([]byte(successContent)))
	}

	expectNotSuccessBody := func(res *http.Response) {
		p := make([]byte, len(successContent))
		_, err := gbytes.TimeoutReader(res.Body, time.Second).Read(p)
		Expect(err).NotTo(HaveOccurred())
		Expect(p).ShouldNot(Equal([]byte(successContent)))
	}

	Context("should succeed", func() {

		It("with no authentication", func() {
			c := ts.Client()
			res, err := c.Get(ts.URL)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusUnauthorized))
			Expect(res.Status).To(Equal("401 Unauthorized"))
			Expect(res.Header.Get("WWW-Authenticate")).To(ContainSubstring(`realm="im-a-test-realm"`))

			expectNotSuccessBody(res)
		})

		It("with valid authentication", func() {
			c := ts.Client()
			r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
			Expect(err).NotTo(HaveOccurred())
			r.SetBasicAuth("test", "valid-pass")
			res, err := c.Do(r)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(res.Status).To(Equal("200 OK"))
			Expect(res.Header.Get("WWW-Authenticate")).To(BeEmpty())

			expectSuccessBody(res)
		})

		It("with invalid authentication", func() {
			c := ts.Client()
			r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
			Expect(err).NotTo(HaveOccurred())
			r.SetBasicAuth("test", "invalid-pass")
			res, err := c.Do(r)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusUnauthorized))
			Expect(res.Status).To(Equal("401 Unauthorized"))
			Expect(res.Header.Get("WWW-Authenticate")).To(ContainSubstring(`realm="im-a-test-realm"`))

			expectNotSuccessBody(res)
		})

	})

	Context("should fail", func() {
	})

})
