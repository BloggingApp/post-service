package dto

type CreatePostRequest struct {
	Title   string   `json:"title" binding:"required,min=2"`
	Content string   `json:"content" binding:"required,min=20"`
	Tags    []string `json:"tags"`
}

type GetPostsRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type EditPostRequest struct {
	ID      int64   `json:"id"`
	Title   *string `json:"title"`
	Content *string `json:"content"`
}
