FROM redis:7.4.12-alpine

COPY cache/redis.conf /usr/local/etc/redis/scrcpycat.conf
COPY cache/licenses/REDIS-LICENSE.txt cache/SOURCE.md /usr/share/licenses/scrcpycat-cache/
COPY LICENSE /usr/share/licenses/scrcpycat-cache/packaging-LICENSE

VOLUME /data
EXPOSE 6379
CMD ["redis-server", "/usr/local/etc/redis/scrcpycat.conf"]
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=20 CMD redis-cli ping | grep -q PONG
ARG VERSION=0.1.1-dev
ARG REVISION=unknown
LABEL org.opencontainers.image.title="ScrcpyCat Cache" \
      org.opencontainers.image.description="Redis message bus with persistence and health checks for the ScrcpyCat stack" \
      org.opencontainers.image.source="https://github.com/ChlorophyTeio/scrcpy-web-docker" \
      org.opencontainers.image.base.name="docker.io/library/redis:7.4.12-alpine" \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.revision=$REVISION
