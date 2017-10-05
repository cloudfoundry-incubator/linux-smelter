package credhub_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"

	. "github.com/cloudfoundry-incubator/credhub-cli/credhub"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Interpolate", func() {
	Describe("InterpolateString()", func() {
		var server *ghttp.Server
		BeforeEach(func() {
			server = ghttp.NewServer()
		})

		Context("when VCAP_SERVICES contains credhub refs", func() {
			var vcapServicesValue string
			BeforeEach(func() {
				vcapServicesValue = `{"my-server":[{"credentials":{"credhub-ref":"(//my-server/creds)"}}]}`
			})

			Context("when credhub successfully interpolates", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/info"),
							ghttp.RespondWith(http.StatusOK, "{}"),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", "/api/v1/interpolate"),
							ghttp.VerifyBody([]byte(vcapServicesValue)),
							ghttp.RespondWith(http.StatusOK, "INTERPOLATED_JSON"),
						))
				})

				It("returns VCAP_SERVICES with the interpolated content", func() {
					interpolated, err := InterpolateString(server.URL(), vcapServicesValue)
					Expect(err).ToNot(HaveOccurred())
					Expect(interpolated).To(Equal("INTERPOLATED_JSON"))
				})
			})

			Context("when credhub fails", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/info"),
							ghttp.RespondWith(http.StatusOK, "{}"),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", "/api/v1/interpolate"),
							ghttp.VerifyBody([]byte(vcapServicesValue)),
							ghttp.RespondWith(http.StatusInternalServerError, "{}"),
						))
				})

				It("returns an error", func() {
					_, err := InterpolateString(server.URL(), vcapServicesValue)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when VCAP_SERVICES does not contain credhub refs", func() {
			It("does not attempt to do any credhub interpolation", func() {
				vcapServicesValue := `{"my-server":[{"credentials":{}}]}`
				interpolated, err := InterpolateString(server.URL(), vcapServicesValue)
				Expect(err).ToNot(HaveOccurred())
				Expect(interpolated).To(Equal(vcapServicesValue))
			})
		})

		Context("when an invalid credhub_uri is used", func() {
			It("returns an error", func() {
				_, err := InterpolateString("reallybadURL://not.real.at.all", `{ "credentials": {"credhub-ref": "doesntmatter"} }`)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("(ch *CredHub) InterpolateString()", func() {
		It("requests to interpolate the VCAP_SERVICES object", func() {
			dummy := &DummyAuth{Response: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString("")),
			}}

			ch, _ := New("https://example.com", Auth(dummy.Builder()), ServerVersion("1.2.3"))

			expectedPayload := `{ "VCAP_SERVICES JSON": "Body goes here" }`
			ch.InterpolateString(expectedPayload)

			urlPath := dummy.Request.URL.Path
			Expect(urlPath).To(Equal("/api/v1/interpolate"))
			Expect(dummy.Request.Method).To(Equal(http.MethodPost))
			Expect(ioutil.ReadAll(dummy.Request.Body)).To(MatchJSON(expectedPayload))
		})

		Context("when successful", func() {
			It("returns the interpolated credential", func() {
				interpolatedResponse := `{ "Your VCAP_SERVICES": "totally-interpolated" }`
				dummy := &DummyAuth{Response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewBufferString(interpolatedResponse)),
				}}

				ch, _ := New("https://example.com", Auth(dummy.Builder()), ServerVersion("1.2.3"))

				interpolatedServices, err := ch.InterpolateString(`{ "VCAP_SERVICES JSON": "Body goes here" }`)
				Expect(err).ToNot(HaveOccurred())

				Expect(interpolatedServices).To(MatchJSON(interpolatedResponse))
			})
		})

		Context("when request fails", func() {
			It("returns an error", func() {
				networkError := errors.New("Network error occurred")
				dummy := &DummyAuth{Error: networkError}
				ch, _ := New("https://example.com", Auth(dummy.Builder()), ServerVersion("1.2.3"))

				_, err := ch.InterpolateString(`{ "whatever": "stuff" }`)

				Expect(err).To(HaveOccurred())
			})
		})

		Context("when vcapServicesBody is invalid", func() {
			It("returns an error", func() {
				dummy := &DummyAuth{Response: &http.Response{
					Body: ioutil.NopCloser(bytes.NewBufferString("")),
				}}

				ch, _ := New("https://example.com", Auth(dummy.Builder()), ServerVersion("1.2.3"))

				_, err := ch.InterpolateString(`{ "VCAP_SERVICES JSON": "missing quote and curly brace`)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
