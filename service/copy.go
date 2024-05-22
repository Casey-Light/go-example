package service

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// Copy - копирует файл или папку. Ошибка по каждому файлу записывается в результат.
func (s *Service) Copy(ctx context.Context, info *http.CallerInfo, req dto.CopyFileRequest) ([]dto.CopyFileResponse, error) {
	var errs error
	copyResponses := make([]dsm.PathResponse, 0, len(req.Files))

	for _, file := range req.Files {
		var result dsm.CopyFileResponse
		result.FromPath = strings.TrimSpace(file.FromPath)
		result.ToPath = strings.TrimSpace(file.ToPath)
		result.Code = 200

		err := s.copyOne(ctx, info, result.FromPath, result.ToPath, req)
		if err != nil {
			if errors.Is(err, errs.NotFound) {
				result.Error = errs.NotFound.Error()
				result.Code = 404
			} else if errors.Is(err, model.ErrInvalidPath) {
				result.Error = model.ErrInvalidPath.Error()
				result.Code = 422
			} else if errors.Is(err, errs.PermissionDenied) {
				result.Error = err.Error()
				result.Code = 403
			} else if errors.Is(err, errs.AlreadyExists) {
				result.Error = errs.AlreadyExists.Error()
				result.Code = 409
			} else if errors.Is(err, model.ErrSharedPath) {
				result.Error = model.ErrSharedPath.Error()
				result.Code = 449
			} else {
				result.Error = errs.ErrInternal.Error()
				result.Code = 500
				s.l.Errorx(err, "copy one", "file", file)
			}

			copyResponses = append(copyResponses, result)
			errs = multierr.Append(errs, err)
			continue
		}

		copyResponses = append(copyResponses, result)
	}

	return copyResponses, errs
}

// copyOne - Возвращает файлы для копирования по переданным путям.
func (s *Service) copyOne(
	ctx context.Context, info *http.CallerInfo,
	copyFrom, copyTo string,
) error {
	if copyFrom == copyTo {
		return errors.Wrap(errs.AlreadyExists, "original and destination paths are the same")
	}

	fileFrom, fileFromAccess, err := s.GetFileByPath(ctx, info, copyFrom)
	if err != nil {
		return errors.Wrap(err, "get fileFrom")
	}

	// Если пытаются копировать папку в саму себя, то выдаём ошибку.
	if fileFrom.IsFolder && strings.Contains(copyTo, fileFrom.Path+"/") {
		return errors.Wrap(errs.ErrInvalidPath, "you can not copy folder to itself")
	}

	// Есть ли права на копирование исходного файла?
	if !helper.Contains(fileFromAccess, accessRights.Read) {
		return errors.Wrap(errs.PermissionDenied, "check read access")
	}

	fileTo, fileToParentsAccess, _, err := s.GetFile(ctx, info, copyTo)
	if err != nil {
		return errors.Wrap(err, "get fileTo and parent")
	}

	// Есть ли права на копирование в папку?
	if !helper.Contains(fileToParentsAccess, accessRights.CreateFile) {
		return errors.Wrap(errs.PermissionDenied, "check create child access")
	}

	err = r.Copy(info.UserId, fileFrom, fileTo)
	return errors.Wrap(err, "copy")
}
