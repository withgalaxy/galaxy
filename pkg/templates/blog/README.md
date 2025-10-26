# {{.ProjectName}}

A modern blog built with Galaxy featuring markdown content, collections, and static generation.

## Features

- ✅ Markdown content with YAML frontmatter
- ✅ Content collections API
- ✅ Syntax highlighting
- ✅ Draft support
- ✅ Component-based architecture

## Commands

```bash
galaxy dev      # Start dev server
galaxy build    # Build for production
galaxy preview  # Preview build
```

## Writing Posts

Create markdown files in `src/content/blog/`:

```markdown
---
layout: "../../layouts/BlogPost.gxc"
title: "My Post"
pubDate: "2024-01-15"
author: "Your Name"
tags: ["go", "web"]
draft: false
---

# Hello World

Your content here...
```

## Learn More

- [Galaxy Documentation](https://galaxy.dev)
- [Markdown Guide](https://galaxy.dev/guides/markdown)
