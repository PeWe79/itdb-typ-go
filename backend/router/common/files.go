package common

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"itdb-backend/internal/common/primitives"
)

func GetMultipartFileHeader(form *multipart.Form, key string) (*multipart.FileHeader, error) {
	if form == nil || form.File == nil {
		return nil, errors.New("missing multipart form")
	}
	files := form.File[key]
	if len(files) == 0 {
		return nil, errors.New("missing file")
	}
	return files[0], nil
}

func SetContentDispositionHeader(w http.ResponseWriter, disposition, displayName string) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = "file"
	}

	fallback := primitives.ContentDispositionFallbackName(name)
	escaped := url.PathEscape(name)
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`, disposition, fallback, escaped),
	)
}
