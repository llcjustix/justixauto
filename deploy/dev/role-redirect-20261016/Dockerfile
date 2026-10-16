# Build only the exact legacy frontend plus its source patch; no application Go.
FROM node:24.21.0-alpine@sha256:ebfe2f90462722a7a4de65e91990e97fe0d401c70e0e762c5b53302f905ec1c1 AS web
WORKDIR /src/web
COPY web ./
RUN npm ci --ignore-scripts --no-audit --no-fund && npm run build:apps

FROM poirot92/justix-auto@sha256:2ad5fd04912005c828e44da9c526582e3ab3e2c2a64776dd767567d8d5ddb811 AS baseline

FROM golang:1.27.1-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS package
WORKDIR /src
ARG OPERATION_SHA
ARG PATCH_SHA
COPY cmd/dev-role-redirect-20261016/main.go /src/install.go
COPY deploy/dev/role-redirect-20261016/legacy-session.patch /src/legacy-session.patch
RUN test "$(sha256sum /src/legacy-session.patch | cut -d' ' -f1)" = "$PATCH_SHA" \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.operationSHA=$OPERATION_SHA -X main.patchSHA=$PATCH_SHA" -o /out/installer /src/install.go
COPY --from=web /src/web/apps/realization/dist /out/payload/realization
COPY --from=web /src/web/apps/financing/dist /out/payload/financing
COPY --from=web /src/web/apps/insurance/dist /out/payload/insurance
COPY --from=web /src/web/apps/admin/dist /out/payload/admin
COPY --from=baseline /app/web/apps/realization/dist/index.html /out/payload/baseline/realization/index.html
COPY --from=baseline /app/web/apps/financing/dist/index.html /out/payload/baseline/financing/index.html
COPY --from=baseline /app/web/apps/insurance/dist/index.html /out/payload/baseline/insurance/index.html
COPY --from=baseline /app/web/apps/admin/dist/index.html /out/payload/baseline/admin/index.html
# These binaries exist only in this build stage, never in the delivered payload.
COPY --from=baseline /app/api /tmp/baseline-api
COPY --from=baseline /app/migrate /tmp/baseline-migrate
RUN sha256sum /tmp/baseline-api > /out/payload/baseline/api.sha256 \
 && sha256sum /tmp/baseline-migrate > /out/payload/baseline/migrate.sha256 \
 && /out/installer manifest \
 && cp /out/installer /out/payload/installer \
 && chmod -R a+rX /out/payload

FROM scratch
COPY --from=package /out/payload /payload
