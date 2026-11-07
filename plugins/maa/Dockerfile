FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS build
WORKDIR /src
COPY plugins/maa/worker/go.mod plugins/maa/worker/go.sum ./
RUN go mod download
COPY plugins/maa/worker ./
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/maa-plugin .
FROM debian:trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends adb bash ca-certificates curl git python3 tini tzdata util-linux libgcc-s1 libstdc++6 libgomp1 libatomic1 && rm -rf /var/lib/apt/lists/* && mkdir -p /opt/android && ln -s /usr/bin/adb /opt/android/adb
ARG TARGETARCH
COPY plugins/maa/docker/install-cli.py /tmp/install-cli.py
RUN TARGETARCH=$TARGETARCH python3 /tmp/install-cli.py && rm /tmp/install-cli.py
COPY --from=build /out/maa-plugin /app/main
COPY plugins/maa/docker/configure-device.py /opt/configure-device.py
COPY --chmod=755 plugins/maa/docker/start.sh /opt/start.sh
COPY --chmod=755 plugins/maa/docker/maa-phone-display /usr/local/bin/maa-phone-display
COPY plugins/maa/resources /opt/maa-plugin/resources
COPY plugins/maa/licenses /usr/share/licenses/maa
COPY LICENSE /usr/share/licenses/maa/plugin-LICENSE
ENV MAA_APP_DIR=/app MAA_CONFIG_DIR=/app/config/maa ADB_SERVER_SOCKET=tcp:scrcpy-web:5037 LANG=C.UTF-8 TZ=Asia/Shanghai
WORKDIR /app
VOLUME /app/config /root/.local/share/maa /root/.cache/maa
EXPOSE 8080
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/bin/tini","-g","--","/opt/start.sh"]
HEALTHCHECK --interval=20s --timeout=5s --start-period=300s --retries=3 CMD curl -fsS http://127.0.0.1:8080/healthz || exit 1
LABEL org.opencontainers.image.title="MAA plugin for ScrcpyCat" org.opencontainers.image.description="Optional independent MAA worker, schedules and core/resource updates"
