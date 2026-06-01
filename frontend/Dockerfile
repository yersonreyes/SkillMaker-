# Stage 1: Build — compila la app Angular con AOT
FROM node:20-alpine AS build
WORKDIR /app

# Cache de dependencias: copiar package*.json antes que el codigo fuente
COPY package*.json ./
RUN npm ci --legacy-peer-deps

# Copiar codigo fuente y compilar para produccion
COPY . .
RUN npm run build:prod

# Stage 2: Runtime — sirve los archivos estaticos con nginx
FROM nginx:alpine AS runtime
COPY --from=build /app/dist/skillmaker-frontend/browser /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
