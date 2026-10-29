# ---- Stage 1: build the Vue app into static files ----
# Install dependencies first, so this layer is reused when only source code changes.
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci || npm install
COPY . .
RUN npm run build

# ---- Stage 2: serve the files with nginx ----
# nginx also forwards /api to the Go container, so the browser talks to one address.
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
