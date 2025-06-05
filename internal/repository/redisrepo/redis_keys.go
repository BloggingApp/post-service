package redisrepo

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	POST_KEY = "post:%d" // <postID>
	AUTHOR_POSTS_KEY = "author:%s-posts:%d:%d" // <authorID>:<limit>:<offset>
	USER_CACHE_KEY = "user-cache:%s" // <userID>
	POST_COMMENTS_KEY = "post:%d-comments:%d:%d" // <postID>:<limit>:<offset>
	COMMENT_REPLIES_KEY = "post:%d-comment:%d-replies:%d:%d" // <postID>:<commentID>:<limit>:<offset>
	USER_LIKES_KEY = "user:%s-likes:%d:%d" // <userID>:<limit>:<offset>
	IS_LIKED_POST_KEY = "user:%s-is-liked-post:%d" // <userID>:<postID>
	POST_LIKES_KEY = "post-likes:%d" // <postID>
	POST_LIKES_KEY_PATTERN = "post-likes:*"
	COMMENT_LIKES_KEY = "comment-likes:%d" // <commentID>
	COMMENT_LIKES_KEY_PATTERN = "comment-likes:*"
	IS_LIKED_COMMENT_KEY = "user:%s-is-liked-comment:%d" // <userID>:<commentID>
	TRENDING_POSTS_KEY = "trending-posts:%d" // <limit>
	SEARCH_POSTS_RESULT_BY_TITLE_KEY = "search-posts-result-by-title:%s:%d:%d" // <title>:<limit>:<offset>
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

func IsLikedPostKey(userID string, postID int64) string {
	return fmt.Sprintf(IS_LIKED_POST_KEY, userID, postID)
}

func PostLikesKey(postID int64) string {
	return fmt.Sprintf(POST_LIKES_KEY, postID)
}

func GetPostIDFromPostLikesKey(key string) (int64, error) {
	parts := strings.Split(key, ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("no part with post ID")
	}
	postID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	return int64(postID), nil
}

func CommentLikesKey(commentID int64) string {
	return fmt.Sprintf(COMMENT_LIKES_KEY, commentID)
}

func GetCommentIDFromCommentLikesKey(key string) (int64, error) {
	parts := strings.Split(key, ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("no part with comment ID")
	}
	commentID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	return int64(commentID), nil
}

func IsLikedCommentKey(userID string, commentID int64) string {
	return fmt.Sprintf(IS_LIKED_COMMENT_KEY, userID, commentID)
}

func TrendingPostsKey(limit int) string {
	return fmt.Sprintf(TRENDING_POSTS_KEY, limit)
}

func SearchPostsResultByTitleKey(title string, limit, offset int) string {
	return fmt.Sprintf(SEARCH_POSTS_RESULT_BY_TITLE_KEY, title, limit, offset)
}
