# Yoyo product site

Source for `https://shenchangxin.github.io/yoyo/`. Vite builds into `../site`.

```bash
npm install
npm run dev
npm run build
```

Copy lives in `src/copy.ts`. Three languages share one narrative spine.

## Version

The build reads the product version from `../internal/version/version.go`. The page uses it for the download links when the GitHub release API cannot be reached. `pages.yml` rebuilds the site after each release, so that fallback is the version just published.

## Screenshots

The originals are the PNG files in `../assets`. The site serves WebP copies from `public/shots`. After replacing an original, convert it again:

```bash
ffmpeg -i ../assets/今天.png -c:v libwebp -quality 88 -preset picture public/shots/today.webp
```

The file names are `today`, `tasks`, `skills`, `video`, `study`, and `idol`. `public/og.jpg` is the link preview, 1200 × 630, cut from the Today shot:

```bash
ffmpeg -i ../assets/今天.png -vf "scale=1200:-1:flags=lanczos,crop=1200:630:0:0" -q:v 3 public/og.jpg
```
