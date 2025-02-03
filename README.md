## post-service

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
- **`[PUB]` GET** -> `/author/:<userID>` - *get `:userID`'s posts*
- **`[AUTH]` GET** -> `/liked` - *get user liked posts*

- **`[PUB]` GET** -> `/:<postID>` - *get post by `:postID`*
- **`[AUTH]` GET** -> `/:<postID>/isLiked` - *get if user has liked the post*
- **`[AUTH]` POST** -> `/:<postID>/like` - *like post*
- **`[AUTH]` DELETE** -> `/:<postID>/unlike` - *unlike post*

`/comments`:
- **`[AUTH]` POST** -> `/` - *create a comment to post*
- **`[PUB]` GET** -> `/:<postID>` - *get `:postID` post comments*
- **`[PUB]` GET** -> `/:<postID>/:<commentID>/replies` - *get `:commentID` comment replies*
- **`[AUTH]` DELETE** -> `/:<postID>/:<commentID>` - *delete `:commentID` comment*
- **`[AUTH]` GET** -> `/:<postID>/:<commentID>/isLiked` - *get if user has liked the comment*
- **`[AUTH]` POST** -> `/:<postID>/:<commentID>/like` - *like comment*
- **`[AUTH]` DELETE** -> `/:<postID>/:<commentID>/unlike` - *unlike comment*
