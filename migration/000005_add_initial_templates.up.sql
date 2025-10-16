-- Add Initial Templates
-- Version: 1.0.0
-- Description: Add commonly used container templates matching backend TemplateConfig structure

-- Frontend Templates
INSERT INTO TEMPLATES (name, template_body, template_config, status) VALUES
-- Vue.js Template
('Vue.js', 'FROM node:{{node_version}}-alpine AS builder
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
{% endif %}',
'{
  "description": "점진적으로 채택 가능한 프론트엔드 프레임워크",
  "categories": ["frontend"],
  "display_order": 2,
  "icon_name": "vue",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "8080"},
    {"name": "env_mode", "label": "환경 모드", "type": "select", "options": ["development", "production"], "default": "production"}
  ],
  "template_env": [
    {"key": "NODE_ENV", "value": "{{env_mode}}", "required": true},
    {"key": "PORT", "value": "{{app_port}}", "required": true}
  ],
  "template_ports": [
    {"internal_port": 8080, "network_type": "http", "description": "Vue server port"}
  ],
  "default_ports": [
    {"internal_port": 8080, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"},
    {"key": "PORT", "value": "8080"}
  ]
}', 'active'),

-- Express.js Template
('Express.js', 'FROM node:{{node_version}}-alpine

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
{% endif %}',
'{
  "description": "Node.js 기반 경량 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 4,
  "icon_name": "express",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "3000"},
    {"name": "node_env", "label": "Node 환경", "type": "select", "options": ["development", "production"], "default": "production"},
    {"name": "process_manager", "label": "프로세스 매니저", "type": "select", "options": ["node", "pm2", "nodemon"], "default": "node"}
  ],
  "template_env": [
    {"key": "NODE_ENV", "value": "{{node_env}}", "required": true},
    {"key": "PORT", "value": "{{app_port}}", "required": true}
  ],
  "template_ports": [
    {"internal_port": 3000, "network_type": "http", "description": "Express server port"}
  ],
  "default_ports": [
    {"internal_port": 3000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"},
    {"key": "PORT", "value": "3000"}
  ]
}', 'active'),

-- NestJS Template
('NestJS', 'FROM node:{{node_version}}-alpine AS builder

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

CMD ["node", "dist/main"]',
'{
  "description": "Node.js 기반 엔터프라이즈급 프레임워크",
  "categories": ["backend"],
  "display_order": 5,
  "icon_name": "nestjs",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "3000"},
    {"name": "node_env", "label": "Node 환경", "type": "select", "options": ["development", "production"], "default": "production"}
  ],
  "template_env": [
    {"key": "NODE_ENV", "value": "{{node_env}}", "required": true},
    {"key": "PORT", "value": "{{app_port}}", "required": true}
  ],
  "template_ports": [
    {"internal_port": 3000, "network_type": "http", "description": "NestJS server port"}
  ],
  "default_ports": [
    {"internal_port": 3000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"},
    {"key": "PORT", "value": "3000"}
  ]
}', 'active'),

-- Go Gin Template
('Go Gin', 'FROM golang:{{go_version}}-alpine AS builder

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

CMD ["./main"]',
'{
  "description": "Go 언어 기반 고성능 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 11,
  "icon_name": "go",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "go_version", "label": "Go 버전", "type": "select", "options": ["1.21", "1.22", "1.23"], "default": "1.23"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "8080"},
    {"name": "gin_mode", "label": "Gin 모드", "type": "select", "options": ["debug", "release"], "default": "release"}
  ],
  "template_env": [
    {"key": "GIN_MODE", "value": "{{gin_mode}}", "required": true},
    {"key": "PORT", "value": "{{app_port}}", "required": true}
  ],
  "template_ports": [
    {"internal_port": 8080, "network_type": "http", "description": "Gin server port"}
  ],
  "default_ports": [
    {"internal_port": 8080, "network_type": "http"}
  ],
  "default_env": [
    {"key": "GIN_MODE", "value": "release"},
    {"key": "PORT", "value": "8080"}
  ]
}', 'active'),

-- Database Templates
-- MySQL Template
('MySQL', 'FROM mysql:{{version}}

ENV MYSQL_CHARACTER_SET_SERVER={{charset}}

RUN echo "[mysqld]" > /etc/mysql/conf.d/custom.cnf && \
    echo "character-set-server={{charset}}" >> /etc/mysql/conf.d/custom.cnf && \
    echo "max_connections={{max_connections}}" >> /etc/mysql/conf.d/custom.cnf

EXPOSE 3306

VOLUME ["/var/lib/mysql"]',
'{
  "description": "가장 널리 사용되는 오픈소스 관계형 데이터베이스",
  "categories": ["database"],
  "display_order": 1,
  "icon_name": "mysql",
  "requires_git": false,
  "version": "1.0",
  "template_options": [
    {"name": "version", "label": "MySQL 버전", "type": "select", "options": ["5.7", "8.0", "8.4"], "default": "8.0"},
    {"name": "root_password", "label": "Root 비밀번호", "type": "password", "options": [], "default": ""},
    {"name": "database_name", "label": "데이터베이스명", "type": "text", "options": [], "default": "myapp"},
    {"name": "charset", "label": "문자셋", "type": "select", "options": ["utf8mb4", "utf8"], "default": "utf8mb4"},
    {"name": "max_connections", "label": "최대 연결 수", "type": "number", "options": [], "default": "50"}
  ],
  "template_env": [
    {"key": "MYSQL_ROOT_PASSWORD", "value": "{{root_password}}", "required": true},
    {"key": "MYSQL_DATABASE", "value": "{{database_name}}", "required": true}
  ],
  "template_ports": [
    {"internal_port": 3306, "network_type": "tcp", "description": "MySQL port"}
  ],
  "template_volumes": [
    {"mount_path": "/var/lib/mysql", "capacity": 10240, "description": "MySQL data directory"}
  ],
  "default_ports": [
    {"internal_port": 3306, "network_type": "tcp"}
  ],
  "default_env": [
    {"key": "MYSQL_DATABASE", "value": "myapp"}
  ]
}', 'active'),

-- PostgreSQL Template
('PostgreSQL', 'FROM postgres:{{version}}

RUN echo "max_connections = {{max_connections}}" >> /usr/share/postgresql/postgresql.conf.sample && \
    echo "shared_buffers = {{shared_buffers}}" >> /usr/share/postgresql/postgresql.conf.sample

EXPOSE 5432

VOLUME ["/var/lib/postgresql/data"]',
'{
  "description": "고급 기능을 갖춘 오픈소스 관계형 데이터베이스",
  "categories": ["database"],
  "display_order": 2,
  "icon_name": "postgresql",
  "requires_git": false,
  "version": "1.0",
  "template_options": [
    {"name": "version", "label": "PostgreSQL 버전", "type": "select", "options": ["14", "16", "17"], "default": "16"},
    {"name": "postgres_password", "label": "Postgres 비밀번호", "type": "password", "options": [], "default": ""},
    {"name": "database_name", "label": "데이터베이스명", "type": "text", "options": [], "default": "myapp"},
    {"name": "max_connections", "label": "최대 연결 수", "type": "number", "options": [], "default": "50"},
    {"name": "shared_buffers", "label": "공유 버퍼 크기", "type": "text", "options": [], "default": "128MB"}
  ],
  "template_env": [
    {"key": "POSTGRES_PASSWORD", "value": "{{postgres_password}}", "required": true},
    {"key": "POSTGRES_DB", "value": "{{database_name}}", "required": true}
  ],
  "template_ports": [
    {"internal_port": 5432, "network_type": "tcp", "description": "PostgreSQL port"}
  ],
  "template_volumes": [
    {"mount_path": "/var/lib/postgresql/data", "capacity": 10240, "description": "PostgreSQL data directory"}
  ],
  "default_ports": [
    {"internal_port": 5432, "network_type": "tcp"}
  ],
  "default_env": [
    {"key": "POSTGRES_DB", "value": "myapp"}
  ]
}', 'active');
