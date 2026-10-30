package response

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

const maxJSONBodyBytes = 16 << 10

var ErrUnsupportedMediaType = errors.New("unsupported media type")

func DecodeJSON(writer http.ResponseWriter, request *http.Request, destination any) error {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return ErrUnsupportedMediaType
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func WriteJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func WriteError(writer http.ResponseWriter, status int, code, message string) {
	WriteJSON(writer, status, map[string]string{"error": message, "code": code})
}

func WriteDecodeError(writer http.ResponseWriter, err error) {
	if errors.Is(err, ErrUnsupportedMediaType) {
		WriteError(writer, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json.")
		return
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		WriteError(writer, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds 16 KiB.")
		return
	}
	WriteError(writer, http.StatusBadRequest, "invalid_json", "Request body must contain one valid JSON object.")
}
