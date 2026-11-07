# Cache component sources and licensing

The cache image packages the unmodified Redis binaries from the Docker Official
Image `redis:7.4.12-alpine`, together with ScrcpyCat's configuration and health check.

- Redis source for this version: https://github.com/redis/redis/tree/7.4.12
- Redis source archive: https://github.com/redis/redis/archive/refs/tags/7.4.12.tar.gz
- Docker Official Image build recipes: https://github.com/docker-library/redis/tree/bddefb3f7467990045de5c67cd0c4d06088496a5/7.4/alpine
- Redis license text is included as `REDIS-LICENSE.txt` in this directory of the image,
  and at `cache/licenses/REDIS-LICENSE.txt` in the repository.

Redis 7.4 has its own licensing terms, reproduced in full in that file; the
repository's Apache-2.0 license applies to this project's packaging and
configuration, and does not replace Redis's license. The underlying image retains
its upstream notices and system-library packages.
