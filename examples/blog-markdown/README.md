# Galaxy Markdown Blog Example

Complete example showcasing Galaxy's markdown content system with collections, layouts, and static generation.

## Features

- ✅ Markdown pages with YAML frontmatter
- ✅ Layout integration
- ✅ Syntax highlighting  
- ✅ Content collections API
- ✅ Draft filtering
- ✅ Component-based design
- ✅ Hot reload for markdown
- ✅ Static site generation

## Structure

```
src/
├── content/
│   └── blog/                    # Content collections
│       ├── getting-started.md
│       ├── advanced-routing.md
│       └── markdown-power.md
├── layouts/
│   └── BlogPost.gxc             # Blog post layout
├── components/
│   ├── Layout.gxc
│   ├── BlogList.gxc
│   └── PostCard.gxc
└── pages/
    ├── index.gxc                # Lists all posts
    └── blog/
        └── first-post.md        # Direct markdown page
```

## Usage

```bash
# Development
cd examples/blog-markdown
galaxy dev

# Production build
galaxy build
```

## Writing Posts

Create `src/content/blog/my-post.md`:

```markdown
---
layout: "../../layouts/BlogPost.gxc"
title: "My Awesome Post"
pubDate: "2024-01-15"
author: "Your Name"
tags: ["go", "galaxy"]
draft: false
---

# Hello World

Your content here with **markdown** formatting!
```

## Content Collections API

Query and filter posts:

```go
import "github.com/withgalaxy/galaxy/pkg/content"

collections := content.NewCollections("./src/content")
posts, _ := collections.GetCollection("blog")

// Filter published posts
for _, post := range posts {
    if !post.GetBool("draft") {
        // Use post
    }
}
```

## Features Demonstrated

- [x] YAML frontmatter parsing
- [x] Automatic syntax highlighting
- [x] Layout system
- [x] Content collections
- [x] Draft filtering
- [x] Dynamic routing
- [x] Component reusability
- [x] Hot module reload
