# Simple CDN

## register an origin first
run (POST) /register SiteIdentifier, OriginURL, APIKey
for example

```
curl --location 'http://localhost:8800/register' \
--form 'SiteIdentifier="github_avatars"' \
--form 'OriginURL="avatars.githubusercontent.com"' \
--form 'APIKey="your_secure_api_key"'
```

## Use (call this url instead of origin url in your app)

```
curl --location 'http://localhost:8800/github_avatars/u/20835893'
```

this is origin url : https://avatars.githubusercontent.com/u/20835893

this is cdn url: http://localhost:8800/github_avatars/u/20835893

````


### A. Linux (Ubuntu/Debian)

```bash
sudo apt-get update
sudo apt-get install -y libvips-dev
````

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

# Run on production

```bash
docker build -t cdn -f Dockerfile .
docker run -d -p 8800:8080  --network share_network -v cdn-db-data:/storage/cdn.db --pid=host --name cdn cdn
```
