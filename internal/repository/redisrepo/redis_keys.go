package redisrepo

import "fmt"

const (
	POST_KEY = "post:%d" // <postID>
	AUTHOR_POSTS_KEY = "author:%s-posts:%d:%d" // <authorID>:<limit>:<offset>
	USER_CACHE_KEY = "user-cache:%s" // <userID>
	POST_COMMENTS_KEY = "post:%d-comments:%d:%d" // <postID>:<limit>:<offset>
	COMMENT_REPLIES_KEY = "post:%d-comment:%d-replies:%d:%d" // <postID>:<commentID>:<limit>:<offset>
	USER_LIKES_KEY = "user:%s-likes:%d:%d" // <userID>:<limit>:<offset>
	IS_LIKED_KEY = "user:%s-is-liked:%d" // <userID>:<postID>
	POST_LIKES_KEY = "post-likes:%d" // <postID>
	POST_LIKES_KEY_PATTERN = "post-likes:*"
)

func PostKey(postID int64) string {
	return fmt.Sprintf(POST_KEY, postID)
}

func AuthorPostsKey(authorID string, limit int, offset int) string {
	return fmt.Sprintf(AUTHOR_POSTS_KEY, authorID, limit, offset)
}

func UserCacheKey(userID string) string {
	return fmt.Sprintf(USER_CACHE_KEY, userID)
}

func PostCommentsKey(postID int64, limit int, offset int) string {
	return fmt.Sprintf(POST_COMMENTS_KEY, postID, limit, offset)
}

func CommentRepliesKey(postID int64, commentID int64, limit int, offset int) string {
	return fmt.Sprintf(COMMENT_REPLIES_KEY, postID, commentID, limit, offset)
}

func UserLikesKey(userID string, limit int, offset int) string {
	return fmt.Sprintf(USER_LIKES_KEY, userID, limit, offset)
}

func IsLikedKey(userID string, postID int64) string {
	return fmt.Sprintf(IS_LIKED_KEY, userID, postID)
}

func PostLikesKey(postID int64) string {
	return fmt.Sprintf(POST_LIKES_KEY, postID)
}
