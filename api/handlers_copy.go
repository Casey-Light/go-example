package api

import "github.com/pkg/errors"

// Copy - Копирует файл
func (h *Handler) Copy(info *http.CallerInfo, c echo.Context) error {
	var req dto.CopyFileRequest
	if err := c.Bind(&req); err != nil {
		err = errors.Wrap(err, "bind")
		return http.OutputJsonMessage(c, 400, err.Error(), info.TraceId)
	}

	results, err := h.service.Copy(c.Request().Context(), info, req)
	if err != nil {
		err = errors.Wrap(err, "Copy")
		if errors.Is(err, errs.NotFound) {
			return http.OutputJsonMessage(c, 404, err.Error(), info.TraceId)
		} else if errors.Is(err, errs.Unprocessable) {
			h.l.InfoW(err.Error())
			return http.OutputJsonMessage(c, 422, err.Error(), info.TraceId)
		}

		h.l.Errx(err.Error())
		return http.OutputJsonMessage(c, 500, errs.Internal.Error(), info.TraceId)
	}
	if len(results) == 0 {
		results = make([]dto.CopyFileResponse, 0)
	}

	return http.OutputJson(c, 200, dto.CopyResponse{
		Results: results,
		TraceId: info.TraceId,
	})
}
