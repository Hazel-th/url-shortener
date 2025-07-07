package save

import (
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage"

	resp "url-shortener/internal/lib/api/response"
)

type Request struct {
	Url   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

const aliasLen = 6

type UrlSaver interface {
	SaveUrl(urlToSave string, alias string) error
}

func New(log *slog.Logger, urlSaver UrlSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			w.WriteHeader(http.StatusBadRequest) // 400 при ошибке декодирования
			render.JSON(w, r, resp.Error("failed to decode request"))

			return
		}

		log.Info("request body decoded", slog.Any("req", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest) // 400 при ошибке валидации
			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}
		alias := req.Alias
		if alias == "" {
			alias = random.NewRandomString(aliasLen)
		}

		err := urlSaver.SaveUrl(req.Url, alias)
		if errors.Is(err, storage.ErrURLExists) {
			log.Info("url already exists", slog.String("url", req.Url))

			w.WriteHeader(http.StatusConflict) // 409 конфликт для существующего alias
			render.JSON(w, r, resp.Error("alias already exists"))

			return
		}
		if err != nil {
			log.Error("failed to add url", sl.Err(err))

			w.WriteHeader(http.StatusInternalServerError) // 500 для прочих ошибок сохранения
			render.JSON(w, r, resp.Error("failed to add url"))

			return
		}

		log.Info("url added", slog.String("url", req.Url))

		responseOK(w, r, alias)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}
