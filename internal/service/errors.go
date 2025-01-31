package service

import "errors"

var (
	ErrInternal = errors.New("internal server error")
	ErrFileMustBeImage = errors.New("file must be an image")
	ErrFileMustHaveAValidExtension = errors.New("file must have a valid extension")
	ErrFailedToUploadPostImageToCDN = errors.New("failed to upload post image to CDN")
	ErrHaveAlreadyLikedThePost = errors.New("you have already liked this post")
	ErrHaveNotLikedThePost = errors.New("you haven't liked this post")
)
