# post-service

#### Please note that the main code under development is in the [dev](https://github.com/BloggingApp/post-service/tree/dev) branch

### API Docs
`/api/v1` - base url

**Headers**:
- **`Authorization`: Beader <ACCESS_TOKEN>**

**Designations**:
- **@** - *requires auth*

`/posts`:
- **@POST** -> *create a post*
- **@GET** -> `/my` - *get my posts*
- **GET** -> `/:<postID>` - *get post by `:postID`*
- **GET** -> `/author/:<userID>` - *get `:userID`'s posts*

`/comments`:
- **@POST** -> *create a comment to post*
- **GET** -> `/:<postID>` - *get `:postID` post comments*
- **GET** -> `/:<postID>/:<commentID>/replies` - *get `:commentID` comment replies*
- **DELETE** -> `/:<postID>/:<commentID>` - *delete `:commentID` comment*
