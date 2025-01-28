package service

import "errors"

var (
	ErrInternal = errors.New("internal server error")
	ErrFileMustBeImage = errors.New("file must be an image")
	ErrFileMustHaveAValidExtension = errors.New("file must have a valid extension")
	ErrFailedToUploadPostImageToCDN = errors.New("failed to upload post image to CDN")
)
