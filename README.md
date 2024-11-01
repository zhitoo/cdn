# Simple CDN

### A. Linux (Ubuntu/Debian)

```bash
sudo apt-get update
sudo apt-get install -y libvips-dev
```

### B. macOS (Using Homebrew)

```bash
brew install vips
```

### C. Alpine Linux

```bash
# Install build tools and dependencies
apk add --no-cache build-base gcc git

# Install libvips dependencies
apk add --no-cache vips-dev
```
# Run app
```bash
make run
```
