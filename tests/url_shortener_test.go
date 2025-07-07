package tests

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/random"
)

const (
	host = "localhost:8082"
)

func TestURLShortener_HappyPath(t *testing.T) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	e := httpexpect.Default(t, u.String())

	e.POST("/url").
		WithJSON(save.Request{
			Url:   gofakeit.URL(),
			Alias: random.NewRandomString(10),
		}).
		WithBasicAuth("myuser", "mypass").
		Expect().
		Status(200).
		JSON().Object().
		ContainsKey("alias")
}

//nolint:funlen
func TestURLShortener_SaveRedirect(t *testing.T) {
	testCases := []struct {
		name         string
		url          string
		alias        string
		expectedCode int
		errorMsg     string
	}{
		{
			name:         "Valid URL",
			url:          gofakeit.URL(),
			alias:        gofakeit.Word() + gofakeit.Word(),
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid URL",
			url:          "invalid_url",
			alias:        gofakeit.Word(),
			expectedCode: http.StatusBadRequest,
			errorMsg:     "field Url is not a valid URL",
		},
		{
			name:         "Empty Alias",
			url:          gofakeit.URL(),
			alias:        "",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := url.URL{
				Scheme: "http",
				Host:   host,
			}

			e := httpexpect.Default(t, u.String())

			req := save.Request{
				Url:   tc.url,
				Alias: tc.alias,
			}

			resp := e.POST("/url").
				WithJSON(req).
				WithBasicAuth("myuser", "mypass").
				Expect().
				Status(tc.expectedCode)

			if tc.expectedCode != http.StatusOK {
				// Ошибочный ответ с 400 проверяем поле error с текстом ошибки
				resp.JSON().Object().Value("error").String().Contains(tc.errorMsg)
				return
			}

			// При успешном ответе проверяем alias
			jsonObj := resp.JSON().Object()

			if tc.alias != "" {
				jsonObj.Value("alias").String().IsEqual(tc.alias)
			} else {
				// Если alias пустой, он сгенерирован - проверяем что не пустой
				jsonObj.Value("alias").String().NotEmpty()
			}

			// Проверяем редирект (твой метод, как он у тебя есть)
			alias := tc.alias
			if alias == "" {
				alias = jsonObj.Value("alias").String().Raw()
			}

			testRedirect(t, alias, tc.url)
		})
	}
}

func testRedirect(t *testing.T, alias string, urlToRedirect string) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   alias,
	}

	redirectedToURL, err := api.GetRedirect(u.String())
	require.NoError(t, err)

	require.Equal(t, urlToRedirect, redirectedToURL)
}
