# ============================================================================
# Stage 1: Build the octoshim Go proxy (fully static binary)
# ============================================================================
FROM golang:1.22-alpine AS go-builder

WORKDIR /build
COPY octoshim/go.mod ./
# go.sum is optional for stdlib-only modules; create empty file if absent
RUN touch go.sum && go mod download

COPY octoshim/ .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" -o octoshim .

# ============================================================================
# Stage 2: Download and verify nektos/act
# ============================================================================
FROM alpine:3.19 AS act-downloader

ARG ACT_VERSION=0.2.65
# SHA256 of act_Linux_x86_64.tar.gz for the pinned version
ARG ACT_SHA256=9f2b52f49d6f204d14e9f04b0e8e52de4a024bbb85a1e6745ed6ecb757d7ba87

RUN apk add --no-cache curl

RUN curl -fsSL \
    "https://github.com/nektos/act/releases/download/v${ACT_VERSION}/act_Linux_x86_64.tar.gz" \
    -o /tmp/act.tar.gz \
  && echo "${ACT_SHA256}  /tmp/act.tar.gz" | sha256sum -c - \
  && tar -xzf /tmp/act.tar.gz -C /usr/local/bin/ act \
  && chmod +x /usr/local/bin/act \
  && rm /tmp/act.tar.gz

# ============================================================================
# Stage 3: Runtime image
# ============================================================================
FROM ubuntu:24.04 AS runtime

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      git \
      curl \
      wget \
      jq \
      unzip \
      zip \
      openssl \
      ca-certificates \
      python3 \
      python3-pip \
      docker.io \
      nodejs \
      npm \
      make \
      build-essential \
 && rm -rf /var/lib/apt/lists/*

# Copy static binaries from builder stages
COPY --from=act-downloader /usr/local/bin/act /usr/local/bin/act
COPY --from=go-builder /build/octoshim /usr/local/bin/octoshim

# Pre-create GitHub Actions filesystem layout
RUN mkdir -p \
      /github/home \
      /github/workflow \
      /github/workspace \
      /github/artifacts \
      /github/output \
      /github/_temp \
      /tmp/runner

# Default act configuration: use the self-hosted runner image for all platforms
RUN mkdir -p /root/.config/act \
 && printf \
      '-P ubuntu-latest=catthehacker/ubuntu:act-latest\n-P ubuntu-22.04=catthehacker/ubuntu:act-22.04\n-P ubuntu-20.04=catthehacker/ubuntu:act-20.04\n' \
      > /root/.config/act/actrc

# Install dispatcher Python dependency for the dispatcher image variant
RUN pip3 install --no-cache-dir --break-system-packages pyyaml==6.0.1

COPY entrypoint/ /entrypoint/
COPY dispatcher/dispatcher.py /bridge/dispatcher.py

RUN chmod +x /entrypoint/bridge.sh /entrypoint/mask-tokens.sh

WORKDIR /github/workspace

ENTRYPOINT ["/entrypoint/bridge.sh"]
