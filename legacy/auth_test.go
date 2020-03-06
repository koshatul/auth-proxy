package legacy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/koshatul/auth-proxy/legacy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("legacy", func() {

	denyAuthFunc := func(username string, password string, r *http.Request) (string, bool) {
		return "", false
	}

	var (
		logcore zapcore.Core
		logger  *zap.Logger
		req     *http.Request
	)

	BeforeEach(func() {
		// logcore, logobs = observer.New(zap.DebugLevel)
		logcore, _ = observer.New(zap.DebugLevel)
		logger = zap.New(logcore)
		req = httptest.NewRequest(http.MethodGet, "/", nil)
	})

	Context("should succeed", func() {

		DescribeTable("with bcrypt password",
			func(username, password, passwordHash string) {
				legacyAuthItems := map[string]legacy.AuthItem{
					username: {
						Username: username,
						Password: passwordHash,
					},
				}

				authFunc := legacy.AuthCheckFunc(logger, legacyAuthItems, denyAuthFunc)
				user, ok := authFunc(username, password, req)
				Expect(user).To(Equal(username))
				Expect(ok).To(BeTrue())
			},
			Entry(
				"totally-secure-password",
				"joey-bloggs", "totally-secure-password", "$2a$15$kbSk7OIgk0vHD4vYgShdMO7uGICkpiATpydRl5GnKrBJuBLcM0.yu",
			),
			Entry(
				"another-password",
				"jason-bloggs", "another-password", "$2a$15$GhI/8ct3YlhHlnJOd2/l8Ot2.BsYc058N/5XD9RAIsM8zGIvp6pPW",
			),
			Entry(
				"ioC3phohShae1yiw5uedaed9beuroaRu",
				"julie-bloggs", "ioC3phohShae1yiw5uedaed9beuroaRu", "$2a$15$kuaray2aouiQbjoJlhYeFuPanlEUN5R/S5qh/lnlJhw5r7.XX82xq",
			),
		)

		DescribeTable("with plain password",
			func(username, password, passwordHash string) {
				legacyAuthItems := map[string]legacy.AuthItem{
					username: {
						Username: username,
						Password: passwordHash,
					},
				}

				authFunc := legacy.AuthCheckFunc(logger, legacyAuthItems, denyAuthFunc)
				user, ok := authFunc(username, password, req)
				Expect(user).To(Equal(username))
				Expect(ok).To(BeTrue())
			},
			Entry(
				"totally-secure-password",
				"joey-bloggs", "totally-secure-password", "totally-secure-password",
			),
			Entry(
				"another-password",
				"jason-bloggs", "another-password", "another-password",
			),
			Entry(
				"ioC3phohShae1yiw5uedaed9beuroaRu",
				"julie-bloggs", "ioC3phohShae1yiw5uedaed9beuroaRu", "ioC3phohShae1yiw5uedaed9beuroaRu",
			),
		)

	})

	Context("should fail", func() {

		DescribeTable("with bcrypt password",
			func(username, password, passwordHash string) {
				legacyAuthItems := map[string]legacy.AuthItem{
					username: {
						Username: username,
						Password: passwordHash,
					},
				}

				authFunc := legacy.AuthCheckFunc(logger, legacyAuthItems, denyAuthFunc)
				user, ok := authFunc(username, fmt.Sprintf("%s2", password), req)
				Expect(user).To(BeEmpty())
				Expect(ok).To(BeFalse())
			},
			Entry(
				"totally-secure-password",
				"joey-bloggs", "totally-secure-password", "$2a$15$kbSk7OIgk0vHD4vYgShdMO7uGICkpiATpydRl5GnKrBJuBLcM0.yu",
			),
			Entry(
				"another-password",
				"jason-bloggs", "another-password", "$2a$15$GhI/8ct3YlhHlnJOd2/l8Ot2.BsYc058N/5XD9RAIsM8zGIvp6pPW",
			),
			Entry(
				"ioC3phohShae1yiw5uedaed9beuroaRu",
				"julie-bloggs", "ioC3phohShae1yiw5uedaed9beuroaRu", "$2a$15$kuaray2aouiQbjoJlhYeFuPanlEUN5R/S5qh/lnlJhw5r7.XX82xq",
			),
			Entry(
				"$2a$15$kuaray2aouiQbjoJlhYeFuPanlEUN5R/S5qh/lnlJhw5r7.XX82xq",
				"julie-bloggs-usecrypt",
				"$2a$15$kuaray2aouiQbjoJlhYeFuPanlEUN5R/S5qh/lnlJhw5r7.XX82xq",
				"$2a$15$kuaray2aouiQbjoJlhYeFuPanlEUN5R/S5qh/lnlJhw5r7.XX82xq",
			),
		)

		DescribeTable("with plain password",
			func(username, password, passwordHash string) {
				legacyAuthItems := map[string]legacy.AuthItem{
					username: {
						Username: username,
						Password: passwordHash,
					},
				}

				authFunc := legacy.AuthCheckFunc(logger, legacyAuthItems, denyAuthFunc)
				user, ok := authFunc(username, fmt.Sprintf("%s2", password), req)
				Expect(user).To(BeEmpty())
				Expect(ok).To(BeFalse())
			},
			Entry(
				"totally-secure-password",
				"joey-bloggs", "totally-secure-password", "totally-secure-password",
			),
			Entry(
				"another-password",
				"jason-bloggs", "another-password", "another-password",
			),
			Entry(
				"ioC3phohShae1yiw5uedaed9beuroaRu",
				"julie-bloggs", "ioC3phohShae1yiw5uedaed9beuroaRu", "ioC3phohShae1yiw5uedaed9beuroaRu",
			),
		)

		DescribeTable("with mixed passwords",
			func(username, password, passwordAttempt, passwordHash string) {
				legacyAuthItems := map[string]legacy.AuthItem{
					username: {
						Username: username,
						Password: passwordHash,
					},
				}

				authFunc := legacy.AuthCheckFunc(logger, legacyAuthItems, denyAuthFunc)
				user, ok := authFunc(username, passwordAttempt, req)
				Expect(user).To(BeEmpty())
				Expect(ok).To(BeFalse())
			},
			Entry(
				"plain password (totally-secure/totally-secure-password)",
				"joey-bloggs", "totally-secure-password", "totally-secure", "totally-secure-password",
			),
			Entry(
				"plain password (ioC3phohShae1yiw5uedaed9beuroaRu/another-password)",
				"jason-bloggs", "another-password", "ioC3phohShae1yiw5uedaed9beuroaRu", "another-password",
			),
			Entry(
				"sending bcrypt hash as password",
				"julie-bloggs",
				"ioC3phohShae1yiw5uedaed9beuroaRu",
				"$2a$15$kuaray2aouiQbjoJlhYeFuPanlEUN5R/S5qh/lnlJhw5r7.XX82xq",
				"$2a$15$kuaray2aouiQbjoJlhYeFuPanlEUN5R/S5qh/lnlJhw5r7.XX82xq",
			),
		)

	})

})
