package redisrepo

import "fmt"

const (
	POST_KEY = "post:%d" // <postID>
	AUTHOR_POSTS_KEY = "author:%s-posts:%d:%d" // <authorID>:<limit>:<offset>
	USER_CACHE_KEY = "user-cache:%s" // <userID>
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
