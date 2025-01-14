package model

type PostHashtag struct {
	PostID int64  `json:"post_id"`
	Tag    string `json:"tag"`
}
