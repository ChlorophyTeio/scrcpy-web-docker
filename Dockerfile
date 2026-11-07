FROM --platform=$BUILDPLATFORM node:24-bookworm-slim AS frontend
WORKDIR /src
COPY frontend/web-app/package*.json ./frontend/web-app/
RUN cd frontend/web-app && npm ci --no-audit --no-fund
COPY frontend ./frontend
COPY LICENSE NOTICE ./
COPY third_party/licenses ./third_party/licenses
COPY tools/frontend-licenses.mjs ./tools/frontend-licenses.mjs
RUN cd frontend/web-app && npm run build

FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY third_party/gadb ./third_party/gadb
RUN go mod download
COPY internal ./internal
COPY backend ./backend
COPY agent ./agent
COPY adb-deployer ./adb-deployer
COPY tools/licenses ./tools/licenses
COPY plugins ./plugins
COPY LICENSE NOTICE /out/licenses/
COPY frontend/LICENSE /out/licenses/frontend/LICENSE
COPY third_party/gadb/LICENSE /out/licenses/gadb/LICENSE
COPY third_party/scrcpy/LICENSE /out/licenses/scrcpy/LICENSE
COPY --from=frontend /src/backend/internal/webui/dist ./backend/internal/webui/dist
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=0.1.0-dev
ARG REVISION=unknown
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X github.com/cwithw/ScrcpyCat/internal/buildinfo.Version=$VERSION -X github.com/cwithw/ScrcpyCat/internal/buildinfo.Revision=$REVISION" -o /out/scrcpycat-controlplane ./backend/cmd/controlplane \
 && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/scrcpycat-agent ./agent/cmd/agent \
 && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/scrcpycat-adb-deployer ./adb-deployer/cmd/adb-deployer \
 && go run ./tools/licenses -out /out/licenses/GO_THIRD_PARTY_LICENSES.txt \
 && mkdir -p /out/plugins && for p in plugins/*/plugin.json; do cp "$p" "/out/plugins/$(basename "$(dirname "$p")").json"; done
COPY --from=frontend /src/backend/internal/webui/dist/THIRD_PARTY_LICENSES.txt /out/licenses/FRONTEND_THIRD_PARTY_LICENSES.txt

FROM debian:trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends adb bash ca-certificates nodejs openssl tini util-linux curl && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/scrcpycat-controlplane /out/scrcpycat-agent /out/scrcpycat-adb-deployer /usr/local/bin/
COPY --from=build /out/licenses /usr/share/licenses/scrcpycat
COPY --from=build /out/plugins /opt/scrcpycat/plugins
COPY bin/agent /opt/scrcpycat/agent
COPY bin/scrcpy-server /opt/scrcpycat/scrcpy
COPY --chmod=755 docker/start.sh /opt/start.sh
COPY --chmod=755 docker/adb-usb-supervisor.sh /opt/adb-usb-supervisor.sh
COPY docker/integrated-agent.cjs /opt/integrated-agent.cjs
COPY docker/tls-proxy.cjs /opt/tls-proxy.cjs
ENV SCRCPYCAT_LISTEN_ADDRESS=:8443 SCRCPYCAT_ASSET_DIR=/data/assets ADB_USB_BINARY=/usr/bin/adb ADB_LIBUSB=0 SCRCPYCAT_AGENT_STATE_DIR=/data/agent-state SCRCPYCAT_INTERNAL_AGENT_ENABLED=true SCRCPYCAT_WEBRTC_UDP_MIN=40000 SCRCPYCAT_WEBRTC_UDP_MAX=40100
EXPOSE 9443 8443 5037 40000-40100/udp
VOLUME /data
ENTRYPOINT ["/usr/bin/tini","-g","--","/opt/start.sh"]
HEALTHCHECK --interval=20s --timeout=5s --start-period=30s --retries=3 CMD curl -fsS http://127.0.0.1:8443/healthz || exit 1
LABEL org.opencontainers.image.title="ScrcpyCat" org.opencontainers.image.description="Web console, USB ADB and integrated agent; optional service plugins"
