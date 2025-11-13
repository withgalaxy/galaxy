# Cloudflare Pages Static Site Example

Example Galaxy app configured for deployment to Cloudflare Pages.

## Build

```bash
galaxy build
```

## Deploy

### Option 1: Wrangler CLI (Recommended)

```bash
npm install -g wrangler
wrangler pages deploy dist
```

### Option 2: Git Integration

1. Push to GitHub/GitLab
2. Connect repo in Cloudflare Pages dashboard
3. Auto-deploy on push

## Learn More

[Cloudflare Deployment Guide](https://github.com/withgalaxy/galaxy/blob/main/docs/cloudflare-deployment.md)
