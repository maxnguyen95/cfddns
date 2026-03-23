# cfddns

Minimal Cloudflare DDNS updater written in Go.

`cfddns` updates a single Cloudflare DNS record (`A` or `AAAA`) to match your current public IP address.
The prebuilt image `maxnguyen95/cfddns:latest` is published automatically via CI/CD, so most users only need Docker or Docker Compose.

## What it does

On each sync cycle, the application:

1. resolves the Cloudflare zone from `CLOUDFLARE_ZONE_NAME`
2. detects the current public IP address
3. looks up the target DNS record by `type + name`
4. creates the record if it does not exist
5. updates the record only when the IP or requested settings have changed

## Requirements

### Cloudflare

You need a Cloudflare API token scoped to the target zone with these permissions:

- `Zone:DNS:Edit`
- `Zone:Zone:Read`

Recommended scope:

- include only the specific zone you want to update

### Docker

For the main usage flow, install:

- Docker

If you want to use Docker Compose, also install:

- Docker Compose plugin (`docker compose`)

## Configuration

Create a `.env` file:

```env
CLOUDFLARE_API_TOKEN=your_token_here
CLOUDFLARE_ZONE_NAME=example.com
CLOUDFLARE_RECORD_NAME=home.example.com

# Optional
CLOUDFLARE_RECORD_TYPE=A
CLOUDFLARE_RECORD_PROXIED=false
CLOUDFLARE_RECORD_TTL=1
CLOUDFLARE_RECORD_COMMENT=Managed by cfddns
SYNC_INTERVAL=5m
HTTP_TIMEOUT=10s
HTTP_USER_AGENT=cfddns/1.0 (+https://github.com/maxnguyen95/cfddns)
```

The same `.env` file can be reused for Docker, Docker Compose, and local source builds.

### Required variables

#### `CLOUDFLARE_API_TOKEN`

Cloudflare API token with permissions:

- `Zone:DNS:Edit`
- `Zone:Zone:Read`

#### `CLOUDFLARE_ZONE_NAME`

The root zone managed in Cloudflare.

Example:

```env
CLOUDFLARE_ZONE_NAME=example.com
```

Do **not** put a subdomain here unless that subdomain is a separate delegated zone in Cloudflare.

#### `CLOUDFLARE_RECORD_NAME`

The full DNS record name you want to manage.

Examples:

```env
CLOUDFLARE_RECORD_NAME=example.com
CLOUDFLARE_RECORD_NAME=home.example.com
CLOUDFLARE_RECORD_NAME=vpn.example.com
```

### Example mapping

If you want to update `home.example.com`, use:

```env
CLOUDFLARE_ZONE_NAME=example.com
CLOUDFLARE_RECORD_NAME=home.example.com
```

If you want to update the zone apex record, use:

```env
CLOUDFLARE_ZONE_NAME=example.com
CLOUDFLARE_RECORD_NAME=example.com
```

### Optional variables

#### `CLOUDFLARE_RECORD_TYPE`

Supported values:

- `A` for IPv4
- `AAAA` for IPv6

Default:

```env
CLOUDFLARE_RECORD_TYPE=A
```

#### `CLOUDFLARE_RECORD_PROXIED`

Whether Cloudflare proxy should be enabled for the record.

Examples:

```env
CLOUDFLARE_RECORD_PROXIED=false
CLOUDFLARE_RECORD_PROXIED=true
```

If unset, the current value is preserved on update.

#### `CLOUDFLARE_RECORD_TTL`

TTL for the DNS record.

Allowed values:

- `1` = automatic
- `60` to `86400`

Default behavior:

- if unset and the record already exists, the current TTL is preserved
- if creating a new record and not otherwise specified, the app uses automatic TTL

#### `CLOUDFLARE_RECORD_COMMENT`

Optional comment stored on the Cloudflare DNS record.

#### `SYNC_INTERVAL`

How often the application checks your public IP when running in continuous mode.

Default:

```env
SYNC_INTERVAL=5m
```

#### `HTTP_TIMEOUT`

HTTP timeout for Cloudflare API and public IP lookup requests.

Default:

```env
HTTP_TIMEOUT=10s
```

#### `HTTP_USER_AGENT`

User-Agent header sent with outbound HTTP requests.

Default:

```env
HTTP_USER_AGENT=cfddns/1.0 (+https://github.com/maxnguyen95/cfddns)
```

## Run with Docker

### 1. Run once

Use one-shot mode to verify your configuration:

```bash
docker run --rm --env-file .env maxnguyen95/cfddns:latest --once
```

### 2. Run continuously

```bash
docker run -d \
  --name cfddns \
  --restart unless-stopped \
  --env-file .env \
  maxnguyen95/cfddns:latest
```

### 3. View logs

```bash
docker logs -f cfddns
```

### 4. Stop and remove the container

```bash
docker stop cfddns
docker rm cfddns
```

## Run with Docker Compose

Create `compose.yaml`:

```yaml
services:
  cfddns:
    image: maxnguyen95/cfddns:latest
    container_name: cfddns
    restart: unless-stopped
    env_file:
      - .env
```

### 1. Start the service

Run in foreground:

```bash
docker compose up
```

Run in background:

```bash
docker compose up -d
```

### 2. Check logs

```bash
docker compose logs -f
```

### 3. Stop the service

```bash
docker compose down
```

## Expected behavior

On a successful run, the application will either:

- create the record if it does not exist
- update the record if it exists but the IP is different
- do nothing if the record is already up to date

Typical log messages:

- `starting DDNS sync loop`
- `dns record created`
- `dns record updated`
- `dns record already up to date`

## Verify that it worked

Check the current public IP:

```bash
curl https://api.ipify.org
```

Check the DNS record:

```bash
dig home.example.com +short
```

or:

```bash
nslookup home.example.com
```

If `CLOUDFLARE_RECORD_PROXIED=true`, DNS lookups may return Cloudflare proxy IPs instead of your origin IP.
For direct DDNS verification, `proxied=false` is usually easier.

## CLI behavior

### `--once`

Run a single sync cycle and exit:

```bash
docker run --rm --env-file .env maxnguyen95/cfddns:latest --once
```

## Common mistakes

### Wrong zone name

Wrong:

```env
CLOUDFLARE_ZONE_NAME=home.example.com
```

Correct:

```env
CLOUDFLARE_ZONE_NAME=example.com
CLOUDFLARE_RECORD_NAME=home.example.com
```

### Missing token permissions

Your token must be able to:

- read the zone
- edit DNS records

### Record name outside the zone

This is invalid:

```env
CLOUDFLARE_ZONE_NAME=example.com
CLOUDFLARE_RECORD_NAME=home.otherdomain.com
```

The record name must be either:

- the zone apex itself
- a subdomain of the zone

### Confusion about proxied records

If the record is proxied through Cloudflare, DNS lookups may not show your router or origin server IP directly.
That is expected behavior.

## Build from source (optional)

This section is only for people who want to inspect, modify, or experiment with the code locally.

### Requirements

- Go `1.23+`
- Git

### 1. Clone the repository

```bash
git clone https://github.com/maxnguyen95/cfddns.git
cd cfddns
```

### 2. Reuse the same `.env` flow

```bash
cp .env.example .env
```

Edit `.env` with your Cloudflare settings.
When running from source or as a local binary, `cfddns` loads `.env` automatically from the current working directory.

### 3. Run from source

```bash
go run ./cmd/cfddns --once
go run ./cmd/cfddns
```

### 4. Build a local binary

```bash
mkdir -p bin
go build -trimpath -o ./bin/cfddns ./cmd/cfddns
```

Run the compiled binary:

```bash
./bin/cfddns --once
./bin/cfddns
```

## License

MIT
