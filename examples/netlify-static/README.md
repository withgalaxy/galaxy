# Galaxy Netlify Static Example

Deploy a Galaxy static site to Netlify.

## Quick Start

```bash
# Build the site
galaxy build

# Deploy to Netlify
netlify deploy --prod
```

## Configuration

The key configuration in `galaxy.config.toml`:

```toml
[output]
type = "static"  # SSG only

[adapter]
name = "netlify"
```

## Build Output

Galaxy creates:

- `dist/` - Static build with HTML, CSS, JS, assets
- `dist/_redirects` - Netlify redirects (SPA fallback)
- `dist/_headers` - Asset caching headers

## Deployment Options

### Option 1: Git Integration (Recommended)

1. Push to GitHub/GitLab/Bitbucket
2. Import project in Netlify dashboard
3. Netlify auto-builds on push using `netlify.toml`

### Option 2: Netlify CLI

```bash
npm i -g netlify-cli
netlify login
netlify init
netlify deploy --prod
```

### Option 3: GitHub Actions

```yaml
name: Deploy
on: [push]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build
        run: |
          go install github.com/withgalaxy/galaxy/cmd/galaxy@latest
          galaxy build
      
      - name: Deploy
        uses: nwtgck/actions-netlify@v2
        with:
          publish-dir: './dist'
          production-deploy: true
        env:
          NETLIFY_AUTH_TOKEN: ${{ secrets.NETLIFY_AUTH_TOKEN }}
          NETLIFY_SITE_ID: ${{ secrets.NETLIFY_SITE_ID }}
```

## Limitations

- **SSG only** - Netlify doesn't support Go serverless functions
- For SSR, use `standalone` adapter with Docker/Railway/Fly.io
- API endpoints won't work

## Learn More

- [Netlify Documentation](https://docs.netlify.com)
- [Galaxy Documentation](https://galaxy.dev/docs)
