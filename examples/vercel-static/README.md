# Galaxy Vercel Static Example

Deploy a Galaxy static site to Vercel.

## Quick Start

```bash
# Build the site
galaxy build

# Deploy to Vercel
vercel deploy
```

## Configuration

The key configuration in `galaxy.config.toml`:

```toml
[output]
type = "static"  # SSG only

[adapter]
name = "vercel"
```

## Build Output

Galaxy creates two outputs:

1. `dist/` - Normal static build
2. `dist/.vercel/output/` - Vercel Build Output API v3 format
   - `config.json` - Deployment configuration
   - `static/` - All built files

## Deployment Options

### Option 1: Vercel CLI (Recommended)

```bash
npm i -g vercel
vercel deploy --prod
```

### Option 2: Git Integration

1. Push to GitHub/GitLab/Bitbucket
2. Import project in Vercel dashboard
3. Vercel auto-detects `.vercel/output/`
4. Deploys automatically on push

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
        uses: amondnet/vercel-action@v20
        with:
          vercel-token: ${{ secrets.VERCEL_TOKEN }}
```

## Limitations

- **SSG only** - Vercel doesn't support Go serverless functions
- For SSR, use `standalone` adapter with Docker/Railway/Fly.io
- API endpoints won't work (use external API or serverless Node.js functions)

## Learn More

- [Vercel Documentation](https://vercel.com/docs)
- [Galaxy Documentation](https://galaxy.dev/docs)
- [Vercel Build Output API](https://vercel.com/docs/build-output-api/v3)
