# post-service

#### Please note that the main code under development is in the [dev](https://github.com/BloggingApp/post-service/tree/dev) branch

### API Docs
`/api/v1` - base url

**Headers**:
- **`Authorization`**: Bearer `<ACCESS_TOKEN>`

**Designations**:
- **`[AUTH]`** - ***requires** auth*
- **`[PUB]`** - ***doesn't** require auth*

`/posts`:
- **`[AUTH]` POST** -> `/` - *create a post*
- **`[AUTH]` GET** -> `/my` - *get my posts*
- **`[PUB]` GET** -> `/:<postID>` - *get post by `:postID`*
- **`[PUB]` GET** -> `/author/:<userID>` - *get `:userID`'s posts*
- **`[AUTH]` GET** -> `/isLiked/:<postID>` - *get if user has liked the post*
- **`[AUTH]` POST** -> `/like/:<postID>` - *like post*
- **`[AUTH]` DELETE** -> `/unlike/:<postID>` - *unlike post*
- **`[AUTH]` GET** -> `/liked` - *get user liked posts*

`/comments`:
- **`[AUTH]` POST** -> `/` - *create a comment to post*
- **`[PUB]` GET** -> `/:<postID>` - *get `:postID` post comments*
- **`[PUB]` GET** -> `/:<postID>/:<commentID>/replies` - *get `:commentID` comment replies*
- **`[AUTH]` DELETE** -> `/:<postID>/:<commentID>` - *delete `:commentID` comment*
