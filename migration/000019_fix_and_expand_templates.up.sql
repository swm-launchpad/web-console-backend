-- Fix Template Syntax and Expand Templates
-- Version: 1.0.0
-- Description: Convert existing templates from Jinja2 to Go template syntax and add new templates

-- =============================================================================
-- PHASE 1: Update Existing Templates (Jinja2 → Go Template Syntax)
-- =============================================================================

-- 1. Vue.js Template
UPDATE TEMPLATES SET
  template_body = 'FROM node:{{ .node_version }}-alpine AS builder
WORKDIR /app

{{ if eq .package_manager "npm" }}
COPY package*.json ./
RUN npm ci
{{ else if eq .package_manager "yarn" }}
COPY package.json yarn.lock ./
RUN yarn install --frozen-lockfile
{{ else }}
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
{{ end }}

COPY . .

{{ if eq .env_mode "production" }}
{{ if eq .package_manager "npm" }}
RUN npm run build
{{ else if eq .package_manager "yarn" }}
RUN yarn build
{{ else }}
RUN pnpm build
{{ end }}

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html

RUN echo ''server { listen {{ .app_port }}; location / { root /usr/share/nginx/html; index index.html; try_files $uri $uri/ /index.html; } }'' > /etc/nginx/conf.d/default.conf

EXPOSE {{ .app_port }}
CMD ["nginx", "-g", "daemon off;"]
{{ else }}
EXPOSE {{ .app_port }}
{{ if eq .package_manager "npm" }}
CMD ["npm", "run", "dev"]
{{ else if eq .package_manager "yarn" }}
CMD ["yarn", "dev"]
{{ else }}
CMD ["pnpm", "dev"]
{{ end }}
{{ end }}'
WHERE template_id = 1;

-- 2. Express.js Template
UPDATE TEMPLATES SET
  template_body = 'FROM node:{{ .node_version }}-alpine

WORKDIR /app

{{ if eq .package_manager "npm" }}
COPY package*.json ./
{{ else if eq .package_manager "yarn" }}
COPY package.json yarn.lock ./
{{ else }}
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm
{{ end }}

{{ if eq .node_env "production" }}
{{ if eq .package_manager "npm" }}
RUN npm ci --only=production
{{ else if eq .package_manager "yarn" }}
RUN yarn install --production --frozen-lockfile
{{ else }}
RUN pnpm install --prod --frozen-lockfile
{{ end }}
{{ else }}
{{ if eq .package_manager "npm" }}
RUN npm ci
{{ else if eq .package_manager "yarn" }}
RUN yarn install --frozen-lockfile
{{ else }}
RUN pnpm install --frozen-lockfile
{{ end }}
{{ end }}

{{ if eq .process_manager "pm2" }}
RUN npm install -g pm2
{{ else if eq .process_manager "nodemon" }}
RUN npm install -g nodemon
{{ end }}

COPY . .

EXPOSE {{ .app_port }}

{{ if eq .process_manager "pm2" }}
CMD ["pm2-runtime", "start", "index.js", "--name", "express-app"]
{{ else if eq .process_manager "nodemon" }}
CMD ["nodemon", "index.js"]
{{ else }}
CMD ["node", "index.js"]
{{ end }}'
WHERE template_id = 2;

-- 3. NestJS Template
UPDATE TEMPLATES SET
  template_body = 'FROM node:{{ .node_version }}-alpine AS builder

WORKDIR /app

{{ if eq .package_manager "npm" }}
COPY package*.json ./
RUN npm ci
{{ else if eq .package_manager "yarn" }}
COPY package.json yarn.lock ./
RUN yarn install --frozen-lockfile
{{ else }}
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
{{ end }}

COPY . .

{{ if eq .package_manager "npm" }}
RUN npm run build
{{ else if eq .package_manager "yarn" }}
RUN yarn build
{{ else }}
RUN pnpm build
{{ end }}

FROM node:{{ .node_version }}-alpine

WORKDIR /app

{{ if eq .package_manager "npm" }}
COPY package*.json ./
RUN npm ci --only=production
{{ else if eq .package_manager "yarn" }}
COPY package.json yarn.lock ./
RUN yarn install --production --frozen-lockfile
{{ else }}
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --prod --frozen-lockfile
{{ end }}

COPY --from=builder /app/dist ./dist

EXPOSE {{ .app_port }}

CMD ["node", "dist/main"]'
WHERE template_id = 3;

-- 4. Go Gin Template
UPDATE TEMPLATES SET
  template_body = 'FROM golang:{{ .go_version }}-alpine AS builder

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

EXPOSE {{ .app_port }}

CMD ["./main"]'
WHERE template_id = 4;

-- 5. MySQL Template
UPDATE TEMPLATES SET
  template_body = 'FROM mysql:{{ .version }}

ENV MYSQL_CHARACTER_SET_SERVER={{ .charset }}

RUN echo "[mysqld]" > /etc/mysql/conf.d/custom.cnf && \\
    echo "character-set-server={{ .charset }}" >> /etc/mysql/conf.d/custom.cnf && \\
    echo "max_connections={{ .max_connections }}" >> /etc/mysql/conf.d/custom.cnf

EXPOSE 3306

VOLUME ["/var/lib/mysql"]'
WHERE template_id = 5;

-- 6. PostgreSQL Template
UPDATE TEMPLATES SET
  template_body = 'FROM postgres:{{ .version }}

RUN echo "max_connections = {{ .max_connections }}" >> /usr/share/postgresql/postgresql.conf.sample && \\
    echo "shared_buffers = {{ .shared_buffers }}" >> /usr/share/postgresql/postgresql.conf.sample

EXPOSE 5432

VOLUME ["/var/lib/postgresql/data"]'
WHERE template_id = 6;

-- =============================================================================
-- PHASE 2: Add New Templates
-- =============================================================================

INSERT INTO TEMPLATES (name, template_body, template_config, status) VALUES

-- Frontend Templates --

-- 7. React
('React',
'FROM node:{{ .node_version }}-alpine AS builder

WORKDIR /app

{{ if eq .package_manager "npm" }}
COPY package*.json ./
RUN npm ci
{{ else if eq .package_manager "yarn" }}
COPY package.json yarn.lock ./
RUN yarn install --frozen-lockfile
{{ else }}
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
{{ end }}

COPY . .

{{ if eq .package_manager "npm" }}
RUN npm run build
{{ else if eq .package_manager "yarn" }}
RUN yarn build
{{ else }}
RUN pnpm build
{{ end }}

FROM nginx:alpine

COPY --from=builder /app/{{ .build_output }} /usr/share/nginx/html

RUN echo ''server { \\
    listen 80; \\
    location / { \\
        root /usr/share/nginx/html; \\
        index index.html; \\
        try_files $uri $uri/ /index.html; \\
    } \\
}'' > /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]',
'{
  "description": "React 기반 SPA 애플리케이션",
  "categories": ["frontend"],
  "display_order": 3,
  "icon_name": "react",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"},
    {"name": "build_output", "label": "빌드 출력 디렉토리", "type": "text", "options": [], "default": "build"}
  ],
  "template_ports": [
    {"internal_port": 80, "network_type": "http", "description": "HTTP port"}
  ],
  "default_ports": [
    {"internal_port": 80, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"}
  ]
}',
'active'),

-- 8. Next.js
('Next.js',
'FROM node:{{ .node_version }}-alpine AS deps

WORKDIR /app

{{ if eq .package_manager "npm" }}
COPY package*.json ./
RUN npm ci
{{ else if eq .package_manager "yarn" }}
COPY package.json yarn.lock ./
RUN yarn install --frozen-lockfile
{{ else }}
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
{{ end }}

FROM node:{{ .node_version }}-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

{{ if eq .package_manager "npm" }}
RUN npm run build
{{ else if eq .package_manager "yarn" }}
RUN yarn build
{{ else }}
RUN pnpm build
{{ end }}

FROM node:{{ .node_version }}-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE {{ .app_port }}

CMD ["node", "server.js"]',
'{
  "description": "Next.js React 프레임워크",
  "categories": ["frontend"],
  "display_order": 6,
  "icon_name": "nextjs",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "3000"}
  ],
  "template_ports": [
    {"internal_port": 3000, "network_type": "http", "description": "Next.js server port"}
  ],
  "default_ports": [
    {"internal_port": 3000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"},
    {"key": "PORT", "value": "3000"}
  ]
}',
'active'),

-- 9. Angular
('Angular',
'FROM node:{{ .node_version }}-alpine AS builder

WORKDIR /app

{{ if eq .package_manager "npm" }}
COPY package*.json ./
RUN npm ci
{{ else if eq .package_manager "yarn" }}
COPY package.json yarn.lock ./
RUN yarn install --frozen-lockfile
{{ else }}
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
{{ end }}

COPY . .

{{ if eq .package_manager "npm" }}
RUN npm run build -- --configuration=production
{{ else if eq .package_manager "yarn" }}
RUN yarn build --configuration=production
{{ else }}
RUN pnpm build --configuration=production
{{ end }}

FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html

RUN echo ''server { \\
    listen 80; \\
    location / { \\
        root /usr/share/nginx/html; \\
        index index.html; \\
        try_files $uri $uri/ /index.html; \\
    } \\
}'' > /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]',
'{
  "description": "Angular 프론트엔드 프레임워크",
  "categories": ["frontend"],
  "display_order": 7,
  "icon_name": "angular",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"}
  ],
  "template_ports": [
    {"internal_port": 80, "network_type": "http", "description": "HTTP port"}
  ],
  "default_ports": [
    {"internal_port": 80, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"}
  ]
}',
'active'),

-- 10. Svelte
('Svelte',
'FROM node:{{ .node_version }}-alpine AS builder

WORKDIR /app

{{ if eq .package_manager "npm" }}
COPY package*.json ./
RUN npm ci
{{ else if eq .package_manager "yarn" }}
COPY package.json yarn.lock ./
RUN yarn install --frozen-lockfile
{{ else }}
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
{{ end }}

COPY . .

{{ if eq .package_manager "npm" }}
RUN npm run build
{{ else if eq .package_manager "yarn" }}
RUN yarn build
{{ else }}
RUN pnpm build
{{ end }}

FROM nginx:alpine

COPY --from=builder /app/{{ .build_output }} /usr/share/nginx/html

RUN echo ''server { \\
    listen 80; \\
    location / { \\
        root /usr/share/nginx/html; \\
        index index.html; \\
        try_files $uri $uri/ /index.html; \\
    } \\
}'' > /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]',
'{
  "description": "Svelte 경량 프론트엔드 프레임워크",
  "categories": ["frontend"],
  "display_order": 8,
  "icon_name": "svelte",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"},
    {"name": "build_output", "label": "빌드 출력 디렉토리", "type": "text", "options": [], "default": "public"}
  ],
  "template_ports": [
    {"internal_port": 80, "network_type": "http", "description": "HTTP port"}
  ],
  "default_ports": [
    {"internal_port": 80, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"}
  ]
}',
'active'),

-- 11. Static HTML
('Static HTML',
'FROM nginx:{{ .nginx_version }}-alpine

COPY . /usr/share/nginx/html

RUN echo ''server { \\
    listen 80; \\
    location / { \\
        root /usr/share/nginx/html; \\
        index index.html index.htm; \\
    } \\
}'' > /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]',
'{
  "description": "정적 HTML/CSS/JS 웹사이트",
  "categories": ["frontend"],
  "display_order": 9,
  "icon_name": "nginx",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "nginx_version", "label": "Nginx 버전", "type": "select", "options": ["1.25", "1.26"], "default": "1.26"}
  ],
  "template_ports": [
    {"internal_port": 80, "network_type": "http", "description": "HTTP port"}
  ],
  "default_ports": [
    {"internal_port": 80, "network_type": "http"}
  ]
}',
'active'),

-- Backend Templates --

-- 12. Spring Boot
('Spring Boot',
'FROM maven:{{ .maven_version }}-eclipse-temurin-{{ .java_version }} AS build

WORKDIR /app

{{ if eq .build_tool "maven" }}
COPY pom.xml ./
RUN mvn dependency:go-offline -B
COPY src ./src
RUN mvn clean package -DskipTests
RUN cp target/*.jar app.jar
{{ else }}
COPY build.gradle settings.gradle gradlew ./
COPY gradle ./gradle
RUN chmod +x gradlew && ./gradlew dependencies --no-daemon
COPY src ./src
RUN ./gradlew clean build -x test --no-daemon
RUN cp build/libs/*.jar app.jar
{{ end }}

FROM eclipse-temurin:{{ .java_version }}-jre-alpine

WORKDIR /app

RUN addgroup -S spring && adduser -S spring -G spring

COPY --from=build /app/app.jar .
RUN chown spring:spring app.jar

USER spring

EXPOSE {{ .app_port }}

ENV JAVA_OPTS="-XX:+UseContainerSupport -XX:MaxRAMPercentage=75.0 -XX:InitialRAMPercentage=50.0"

HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:{{ .app_port }}/actuator/health || exit 1

ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS -jar app.jar"]',
'{
  "description": "Java 기반 엔터프라이즈급 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 10,
  "icon_name": "spring",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "java_version", "label": "Java 버전", "type": "select", "options": ["17", "21"], "default": "17"},
    {"name": "build_tool", "label": "빌드 도구", "type": "select", "options": ["maven", "gradle"], "default": "gradle"},
    {"name": "maven_version", "label": "Maven 버전", "type": "select", "options": ["3.9"], "default": "3.9"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "8080"}
  ],
  "template_ports": [
    {"internal_port": 8080, "network_type": "http", "description": "Spring Boot server port"}
  ],
  "default_ports": [
    {"internal_port": 8080, "network_type": "http"}
  ],
  "default_env": [
    {"key": "SPRING_PROFILES_ACTIVE", "value": "prod"},
    {"key": "SERVER_PORT", "value": "8080"}
  ]
}',
'active'),

-- 13. FastAPI
('FastAPI',
'FROM python:{{ .python_version }}-slim AS builder

WORKDIR /app

{{ if eq .dependency_manager "pip" }}
COPY requirements.txt ./
RUN pip install --no-cache-dir --user -r requirements.txt
{{ else if eq .dependency_manager "poetry" }}
RUN pip install --no-cache-dir poetry
COPY pyproject.toml poetry.lock* ./
RUN poetry config virtualenvs.create false && poetry install --no-dev --no-root
{{ else }}
RUN pip install --no-cache-dir pipenv
COPY Pipfile Pipfile.lock* ./
RUN pipenv install --system --deploy
{{ end }}

FROM python:{{ .python_version }}-slim

WORKDIR /app

RUN addgroup --system --gid 1001 app && \
    adduser --system --uid 1001 --gid 1001 app

{{ if eq .dependency_manager "pip" }}
COPY --from=builder /root/.local /root/.local
ENV PATH=/root/.local/bin:$PATH
{{ end }}

COPY . .

RUN chown -R app:app /app

USER app

EXPOSE {{ .app_port }}

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "{{ .app_port }}"]',
'{
  "description": "Python FastAPI 비동기 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 12,
  "icon_name": "fastapi",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "python_version", "label": "Python 버전", "type": "select", "options": ["3.9", "3.10", "3.11", "3.12"], "default": "3.11"},
    {"name": "dependency_manager", "label": "의존성 매니저", "type": "select", "options": ["pip", "poetry", "pipenv"], "default": "pip"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "8000"}
  ],
  "template_ports": [
    {"internal_port": 8000, "network_type": "http", "description": "FastAPI server port"}
  ],
  "default_ports": [
    {"internal_port": 8000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "PYTHONUNBUFFERED", "value": "1"},
    {"key": "PORT", "value": "8000"}
  ]
}',
'active'),

-- 14. Flask
('Flask',
'FROM python:{{ .python_version }}-slim

WORKDIR /app

{{ if eq .dependency_manager "pip" }}
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
{{ else if eq .dependency_manager "poetry" }}
RUN pip install --no-cache-dir poetry
COPY pyproject.toml poetry.lock* ./
RUN poetry config virtualenvs.create false && poetry install --no-dev --no-root
{{ else }}
RUN pip install --no-cache-dir pipenv
COPY Pipfile Pipfile.lock* ./
RUN pipenv install --system --deploy
{{ end }}

COPY . .

RUN addgroup --system --gid 1001 app && \
    adduser --system --uid 1001 --gid 1001 app && \
    chown -R app:app /app

USER app

EXPOSE {{ .app_port }}

CMD ["flask", "run", "--host=0.0.0.0", "--port={{ .app_port }}"]',
'{
  "description": "Python Flask 경량 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 13,
  "icon_name": "flask",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "python_version", "label": "Python 버전", "type": "select", "options": ["3.9", "3.10", "3.11", "3.12"], "default": "3.11"},
    {"name": "dependency_manager", "label": "의존성 매니저", "type": "select", "options": ["pip", "poetry", "pipenv"], "default": "pip"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "5000"}
  ],
  "template_ports": [
    {"internal_port": 5000, "network_type": "http", "description": "Flask server port"}
  ],
  "default_ports": [
    {"internal_port": 5000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "FLASK_APP", "value": "app.py"},
    {"key": "FLASK_ENV", "value": "production"}
  ]
}',
'active'),

-- 15. Django
('Django',
'FROM python:{{ .python_version }}-slim AS builder

WORKDIR /app

{{ if eq .dependency_manager "pip" }}
COPY requirements.txt ./
RUN pip install --no-cache-dir --user -r requirements.txt
{{ else if eq .dependency_manager "poetry" }}
RUN pip install --no-cache-dir poetry
COPY pyproject.toml poetry.lock* ./
RUN poetry config virtualenvs.create false && poetry install --no-dev --no-root
{{ else }}
RUN pip install --no-cache-dir pipenv
COPY Pipfile Pipfile.lock* ./
RUN pipenv install --system --deploy
{{ end }}

FROM python:{{ .python_version }}-slim

WORKDIR /app

{{ if eq .dependency_manager "pip" }}
COPY --from=builder /root/.local /root/.local
ENV PATH=/root/.local/bin:$PATH
{{ end }}

COPY . .

RUN python manage.py collectstatic --noinput

RUN addgroup --system --gid 1001 app && \
    adduser --system --uid 1001 --gid 1001 app && \
    chown -R app:app /app

USER app

EXPOSE {{ .app_port }}

CMD ["gunicorn", "--bind", "0.0.0.0:{{ .app_port }}", "--workers", "4", "config.wsgi:application"]',
'{
  "description": "Python Django 풀스택 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 14,
  "icon_name": "django",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "python_version", "label": "Python 버전", "type": "select", "options": ["3.9", "3.10", "3.11", "3.12"], "default": "3.11"},
    {"name": "dependency_manager", "label": "의존성 매니저", "type": "select", "options": ["pip", "poetry", "pipenv"], "default": "pip"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "8000"}
  ],
  "template_ports": [
    {"internal_port": 8000, "network_type": "http", "description": "Django server port"}
  ],
  "default_ports": [
    {"internal_port": 8000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "DJANGO_SETTINGS_MODULE", "value": "config.settings"},
    {"key": "PYTHONUNBUFFERED", "value": "1"}
  ]
}',
'active'),

-- 16. Laravel
('Laravel',
'FROM php:{{ .php_version }}-fpm-alpine AS builder

WORKDIR /app

RUN apk add --no-cache \
    libpng-dev \
    libjpeg-turbo-dev \
    freetype-dev \
    zip \
    unzip

RUN docker-php-ext-configure gd --with-freetype --with-jpeg && \
    docker-php-ext-install -j$(nproc) gd pdo pdo_mysql

COPY --from=composer:latest /usr/bin/composer /usr/bin/composer

COPY composer.json composer.lock ./
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist

COPY . .

RUN composer dump-autoload --optimize

FROM nginx:alpine

COPY --from=builder /app /var/www/html

COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]',
'{
  "description": "Laravel PHP 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 15,
  "icon_name": "laravel",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "php_version", "label": "PHP 버전", "type": "select", "options": ["8.1", "8.2", "8.3"], "default": "8.2"}
  ],
  "template_ports": [
    {"internal_port": 80, "network_type": "http", "description": "HTTP port"}
  ],
  "default_ports": [
    {"internal_port": 80, "network_type": "http"}
  ],
  "default_env": [
    {"key": "APP_ENV", "value": "production"},
    {"key": "APP_DEBUG", "value": "false"}
  ]
}',
'active'),

-- 17. Ruby on Rails
('Ruby on Rails',
'FROM ruby:{{ .ruby_version }}-alpine AS builder

WORKDIR /app

RUN apk add --no-cache \
    build-base \
    postgresql-dev \
    nodejs \
    yarn

COPY Gemfile Gemfile.lock ./
RUN bundle config set --local deployment true && \
    bundle config set --local without development test && \
    bundle install

FROM ruby:{{ .ruby_version }}-alpine

WORKDIR /app

RUN apk add --no-cache \
    postgresql-client \
    nodejs \
    tzdata

COPY --from=builder /usr/local/bundle /usr/local/bundle
COPY . .

RUN addgroup -S rails && adduser -S rails -G rails && \
    chown -R rails:rails /app

USER rails

EXPOSE {{ .app_port }}

CMD ["rails", "server", "-b", "0.0.0.0", "-p", "{{ .app_port }}"]',
'{
  "description": "Ruby on Rails 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 16,
  "icon_name": "rails",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "ruby_version", "label": "Ruby 버전", "type": "select", "options": ["3.1", "3.2", "3.3"], "default": "3.2"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "3000"}
  ],
  "template_ports": [
    {"internal_port": 3000, "network_type": "http", "description": "Rails server port"}
  ],
  "default_ports": [
    {"internal_port": 3000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "RAILS_ENV", "value": "production"},
    {"key": "RAILS_LOG_TO_STDOUT", "value": "true"}
  ]
}',
'active'),

-- 18. Rust (Actix-web)
('Rust',
'FROM rust:{{ .rust_version }}-alpine AS builder

WORKDIR /app

RUN apk add --no-cache musl-dev

COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

COPY src ./src

RUN cargo build --release

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates

RUN addgroup -S rust && adduser -S rust -G rust

COPY --from=builder /app/target/release/{{ .app_name }} .

RUN chown rust:rust {{ .app_name }}

USER rust

EXPOSE {{ .app_port }}

CMD ["./{{ .app_name }}"]',
'{
  "description": "Rust 고성능 웹 애플리케이션",
  "categories": ["backend"],
  "display_order": 17,
  "icon_name": "rust",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "rust_version", "label": "Rust 버전", "type": "select", "options": ["1.75", "1.76"], "default": "1.76"},
    {"name": "app_name", "label": "애플리케이션 이름", "type": "text", "options": [], "default": "app"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "8080"}
  ],
  "template_ports": [
    {"internal_port": 8080, "network_type": "http", "description": "HTTP port"}
  ],
  "default_ports": [
    {"internal_port": 8080, "network_type": "http"}
  ],
  "default_env": [
    {"key": "RUST_LOG", "value": "info"}
  ]
}',
'active'),

-- 19. .NET Core
('.NET Core',
'FROM mcr.microsoft.com/dotnet/sdk:{{ .dotnet_version }}-alpine AS build

WORKDIR /app

COPY *.csproj ./
RUN dotnet restore

COPY . ./
RUN dotnet publish -c Release -o out

FROM mcr.microsoft.com/dotnet/aspnet:{{ .dotnet_version }}-alpine

WORKDIR /app

RUN addgroup -S dotnet && adduser -S dotnet -G dotnet

COPY --from=build /app/out .

RUN chown -R dotnet:dotnet /app

USER dotnet

EXPOSE {{ .app_port }}

ENTRYPOINT ["dotnet", "{{ .app_name }}.dll"]',
'{
  "description": ".NET Core 크로스플랫폼 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 18,
  "icon_name": "dotnet",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "dotnet_version", "label": ".NET 버전", "type": "select", "options": ["7.0", "8.0"], "default": "8.0"},
    {"name": "app_name", "label": "애플리케이션 이름", "type": "text", "options": [], "default": "App"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "5000"}
  ],
  "template_ports": [
    {"internal_port": 5000, "network_type": "http", "description": "ASP.NET server port"}
  ],
  "default_ports": [
    {"internal_port": 5000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "ASPNETCORE_ENVIRONMENT", "value": "Production"},
    {"key": "ASPNETCORE_URLS", "value": "http://+:5000"}
  ]
}',
'active'),

-- 20. Kotlin Spring Boot
('Kotlin Spring Boot',
'FROM gradle:{{ .gradle_version }}-jdk{{ .java_version }}-alpine AS build

WORKDIR /app

COPY build.gradle.kts settings.gradle.kts ./
COPY gradle ./gradle

RUN gradle dependencies --no-daemon

COPY src ./src

RUN gradle clean build -x test --no-daemon

RUN cp build/libs/*.jar app.jar

FROM eclipse-temurin:{{ .java_version }}-jre-alpine

WORKDIR /app

RUN addgroup -S spring && adduser -S spring -G spring

COPY --from=build /app/app.jar .

RUN chown spring:spring app.jar

USER spring

EXPOSE {{ .app_port }}

ENV JAVA_OPTS="-XX:+UseContainerSupport -XX:MaxRAMPercentage=75.0"

CMD ["sh", "-c", "java $JAVA_OPTS -jar app.jar"]',
'{
  "description": "Kotlin Spring Boot 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 19,
  "icon_name": "kotlin",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "java_version", "label": "Java 버전", "type": "select", "options": ["17", "21"], "default": "17"},
    {"name": "gradle_version", "label": "Gradle 버전", "type": "select", "options": ["8.5"], "default": "8.5"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "8080"}
  ],
  "template_ports": [
    {"internal_port": 8080, "network_type": "http", "description": "Spring Boot server port"}
  ],
  "default_ports": [
    {"internal_port": 8080, "network_type": "http"}
  ],
  "default_env": [
    {"key": "SPRING_PROFILES_ACTIVE", "value": "prod"}
  ]
}',
'active'),

-- 21. Fiber (Go)
('Fiber',
'FROM golang:{{ .go_version }}-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags ''-w -s'' -o main .

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -S fiber && adduser -S fiber -G fiber

COPY --from=builder /build/main .

RUN chown fiber:fiber main

USER fiber

EXPOSE {{ .app_port }}

CMD ["./main"]',
'{
  "description": "Go Fiber 고속 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 20,
  "icon_name": "go",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "go_version", "label": "Go 버전", "type": "select", "options": ["1.21", "1.22", "1.23"], "default": "1.23"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "3000"}
  ],
  "template_ports": [
    {"internal_port": 3000, "network_type": "http", "description": "Fiber server port"}
  ],
  "default_ports": [
    {"internal_port": 3000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "FIBER_ENV", "value": "production"},
    {"key": "PORT", "value": "3000"}
  ]
}',
'active'),

-- Database Templates --

-- 22. MongoDB
('MongoDB',
'FROM mongo:{{ .mongo_version }}

EXPOSE {{ .mongo_port }}

VOLUME ["/data/db", "/data/configdb"]',
'{
  "description": "MongoDB NoSQL 문서 데이터베이스",
  "categories": ["database"],
  "display_order": 21,
  "icon_name": "mongodb",
  "requires_git": false,
  "version": "1.0",
  "template_options": [
    {"name": "mongo_version", "label": "MongoDB 버전", "type": "select", "options": ["6.0", "7.0"], "default": "7.0"},
    {"name": "mongo_port", "label": "MongoDB 포트", "type": "number", "options": [], "default": "27017"}
  ],
  "template_ports": [
    {"internal_port": 27017, "network_type": "tcp", "description": "MongoDB port"}
  ],
  "default_ports": [
    {"internal_port": 27017, "network_type": "tcp"}
  ],
  "default_env": [
    {"key": "MONGO_INITDB_ROOT_USERNAME", "value": "admin"},
    {"key": "MONGO_INITDB_ROOT_PASSWORD", "value": ""}
  ],
  "template_volumes": [
    {"mount_path": "/data/db", "capacity": 10240, "description": "MongoDB data directory"}
  ]
}',
'active'),

-- 23. Redis
('Redis',
'FROM redis:{{ .redis_version }}-alpine

EXPOSE {{ .redis_port }}

VOLUME ["/data"]

CMD ["redis-server", "--port", "{{ .redis_port }}", "--appendonly", "yes"]',
'{
  "description": "Redis 인메모리 데이터 스토어",
  "categories": ["database"],
  "display_order": 22,
  "icon_name": "redis",
  "requires_git": false,
  "version": "1.0",
  "template_options": [
    {"name": "redis_version", "label": "Redis 버전", "type": "select", "options": ["7.0", "7.2"], "default": "7.2"},
    {"name": "redis_port", "label": "Redis 포트", "type": "number", "options": [], "default": "6379"}
  ],
  "template_ports": [
    {"internal_port": 6379, "network_type": "tcp", "description": "Redis port"}
  ],
  "default_ports": [
    {"internal_port": 6379, "network_type": "tcp"}
  ],
  "default_env": [
    {"key": "REDIS_PASSWORD", "value": ""}
  ],
  "template_volumes": [
    {"mount_path": "/data", "capacity": 5120, "description": "Redis data directory"}
  ]
}',
'active');
