package dto

type CreateCommentDto struct {
	PostID   int64  `json:"post_id" binding:"required"`
	ParentID *int64 `json:"parent_id"`
	Content  string `json:"content" binding:"required,min=1"`
}

type GetCommentsDto struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
