# syntax=docker/dockerfile:1

# The build toolchain never reaches the final image: nginx serves the static
# bundle produced in the build stage.

ARG NODE_VERSION=20
ARG NGINX_VERSION=1.27

# --- dependencies -------------------------------------------------------------

FROM node:${NODE_VERSION}-alpine AS deps

WORKDIR /app

# Manifests only, so `npm install` is cached independently of the source.
COPY package.json package-lock.json* ./

# npm ci is reproducible but needs a lockfile; fall back so a fresh clone that
# has not committed one still builds.
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

# --- build --------------------------------------------------------------------

FROM deps AS build

COPY . .

# `npm run build` is `tsc --noEmit && vite build`, so a type error fails the
# image build rather than shipping a broken bundle.
RUN npm run build

# --- runtime ------------------------------------------------------------------

FROM nginx:${NGINX_VERSION}-alpine AS runtime

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

# busybox wget ships with the alpine image.
HEALTHCHECK --interval=30s --timeout=3s --start-period=3s --retries=3 \
    CMD wget --quiet --spider --tries=1 http://127.0.0.1/ || exit 1

# The base image's own entrypoint handles config templating and starts nginx.
