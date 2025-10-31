-- Rollback: Fix Template Syntax and Expand Templates
-- Version: 1.0.0
-- Description: Restore original Jinja2 templates and remove newly added templates

-- =============================================================================
-- PHASE 1: Delete Newly Added Templates
-- =============================================================================

DELETE FROM TEMPLATES WHERE name IN (
  'React',
  'Next.js',
  'Angular',
  'Svelte',
  'Static HTML',
  'Spring Boot',
  'FastAPI',
  'Flask',
  'Django',
  'Laravel',
  'Ruby on Rails',
  'Rust',
  '.NET Core',
  'Kotlin Spring Boot',
  'Fiber',
  'MongoDB',
  'Redis'
);

-- =============================================================================
-- PHASE 2: Restore Original Templates (Go Template → Jinja2)
-- =============================================================================

-- 1. Vue.js Template (Restore Original)
UPDATE TEMPLATES SET
  template_body = 'FROM node:{{node_version}}-alpine AS builder
WORKDIR /app

{% if package_manager == ''npm'' %}
COPY package*.json ./
{% elif package_manager == ''yarn'' %}
COPY package.json yarn.lock* ./
{% else %}
COPY package.json pnpm-lock.yaml* ./
RUN npm install -g pnpm
{% endif %}

{% if package_manager == ''npm'' %}
RUN npm install
{% elif package_manager == ''yarn'' %}
RUN yarn install --frozen-lockfile
{% else %}
RUN pnpm install --frozen-lockfile
{% endif %}

COPY . .

{% if env_mode == ''production'' %}
{% if package_manager == ''npm'' %}
RUN npm run build
{% elif package_manager == ''yarn'' %}
RUN yarn build
{% else %}
RUN pnpm build
{% endif %}

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html

RUN echo ''server { listen {{app_port}}; location / { root /usr/share/nginx/html; index index.html; try_files $uri $uri/ /index.html; } }'' > /etc/nginx/conf.d/default.conf

EXPOSE {{app_port}}
CMD ["nginx", "-g", "daemon off;"]
{% else %}
EXPOSE {{app_port}}
{% if package_manager == ''npm'' %}
CMD ["npm", "run", "serve"]
{% elif package_manager == ''yarn'' %}
CMD ["yarn", "serve"]
{% else %}
CMD ["pnpm", "serve"]
{% endif %}
{% endif %}'
WHERE template_id = 1;

-- 2. Express.js Template (Restore Original)
UPDATE TEMPLATES SET
  template_body = 'FROM node:{{node_version}}-alpine

WORKDIR /app

{% if package_manager == ''npm'' %}
COPY package*.json ./
{% elif package_manager == ''yarn'' %}
COPY package.json yarn.lock* ./
{% else %}
COPY package.json pnpm-lock.yaml* ./
RUN npm install -g pnpm
{% endif %}

{% if node_env == ''production'' %}
{% if package_manager == ''npm'' %}
RUN npm install --only=production
{% elif package_manager == ''yarn'' %}
RUN yarn install --production --frozen-lockfile
{% else %}
RUN pnpm install --prod --frozen-lockfile
{% endif %}
{% else %}
{% if package_manager == ''npm'' %}
RUN npm install
{% elif package_manager == ''yarn'' %}
RUN yarn install
{% else %}
RUN pnpm install
{% endif %}
{% endif %}

{% if process_manager == ''pm2'' %}
RUN npm install -g pm2
{% elif process_manager == ''nodemon'' %}
RUN npm install -g nodemon
{% endif %}

COPY . .

EXPOSE {{app_port}}

{% if process_manager == ''pm2'' %}
CMD ["pm2-runtime", "start", "index.js", "--name", "express-app"]
{% elif process_manager == ''nodemon'' %}
CMD ["nodemon", "index.js"]
{% else %}
CMD ["node", "index.js"]
{% endif %}'
WHERE template_id = 2;

-- 3. NestJS Template (Restore Original)
UPDATE TEMPLATES SET
  template_body = 'FROM node:{{node_version}}-alpine AS builder

WORKDIR /app

{% if package_manager == ''npm'' %}
COPY package*.json ./
{% elif package_manager == ''yarn'' %}
COPY package.json yarn.lock* ./
{% else %}
COPY package.json pnpm-lock.yaml* ./
RUN npm install -g pnpm
{% endif %}

{% if package_manager == ''npm'' %}
RUN npm install
{% elif package_manager == ''yarn'' %}
RUN yarn install --frozen-lockfile
{% else %}
RUN pnpm install --frozen-lockfile
{% endif %}

COPY . .

{% if package_manager == ''npm'' %}
RUN npm run build
{% elif package_manager == ''yarn'' %}
RUN yarn build
{% else %}
RUN pnpm build
{% endif %}

FROM node:{{node_version}}-alpine

WORKDIR /app

{% if package_manager == ''npm'' %}
COPY package*.json ./
RUN npm install --only=production
{% elif package_manager == ''yarn'' %}
COPY package.json yarn.lock* ./
RUN yarn install --production --frozen-lockfile
{% else %}
COPY package.json pnpm-lock.yaml* ./
RUN npm install -g pnpm && pnpm install --prod --frozen-lockfile
{% endif %}

COPY --from=builder /app/dist ./dist

EXPOSE {{app_port}}

CMD ["node", "dist/main"]'
WHERE template_id = 3;

-- 4. Go Gin Template (Restore Original)
UPDATE TEMPLATES SET
  template_body = 'FROM golang:{{go_version}}-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /build/main .

EXPOSE {{app_port}}

CMD ["./main"]'
WHERE template_id = 4;

-- 5. MySQL Template (Restore Original)
UPDATE TEMPLATES SET
  template_body = 'FROM mysql:{{version}}

ENV MYSQL_CHARACTER_SET_SERVER={{charset}}

RUN echo "[mysqld]" > /etc/mysql/conf.d/custom.cnf && \
    echo "character-set-server={{charset}}" >> /etc/mysql/conf.d/custom.cnf && \
    echo "max_connections={{max_connections}}" >> /etc/mysql/conf.d/custom.cnf

EXPOSE 3306

VOLUME ["/var/lib/mysql"]'
WHERE template_id = 5;

-- 6. PostgreSQL Template (Restore Original)
UPDATE TEMPLATES SET
  template_body = 'FROM postgres:{{version}}

RUN echo "max_connections = {{max_connections}}" >> /usr/share/postgresql/postgresql.conf.sample && \
    echo "shared_buffers = {{shared_buffers}}" >> /usr/share/postgresql/postgresql.conf.sample

EXPOSE 5432

VOLUME ["/var/lib/postgresql/data"]'
WHERE template_id = 6;
