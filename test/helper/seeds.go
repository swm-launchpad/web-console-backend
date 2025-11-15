package helper

import (
	"database/sql"
	"fmt"
)

// SeedTemplates inserts template data for testing
func SeedTemplates(db *sql.DB) error {
	// Template data for testing purposes
	// Note: Production templates are managed in swm-launchpad/container-go-template repository
	templates := []struct {
		id     int
		name   string
		body   string
		config string
		status string
	}{
		{
			id:   1,
			name: "Vue.js",
			body: `FROM node:{{ .node_version }}-alpine AS builder
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

RUN echo 'server { listen {{ .app_port }}; location / { root /usr/share/nginx/html; index index.html; try_files $uri $uri/ /index.html; } }' > /etc/nginx/conf.d/default.conf

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
{{ end }}`,
			config: `{
  "description": "Vue.js 프론트엔드 프레임워크",
  "categories": ["frontend"],
  "display_order": 1,
  "icon_name": "vue",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"},
    {"name": "env_mode", "label": "환경 모드", "type": "select", "options": ["development", "production"], "default": "production"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "80"}
  ],
  "default_ports": [
    {"internal_port": 80, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"}
  ]
}`,
			status: "active",
		},
		{
			id:   2,
			name: "Express.js",
			body: `FROM node:{{ .node_version }}-alpine

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
{{ end }}`,
			config: `{
  "description": "Express.js Node.js 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 2,
  "icon_name": "express",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"},
    {"name": "node_env", "label": "Node 환경", "type": "select", "options": ["development", "production"], "default": "production"},
    {"name": "process_manager", "label": "프로세스 매니저", "type": "select", "options": ["none", "pm2", "nodemon"], "default": "none"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "3000"}
  ],
  "default_ports": [
    {"internal_port": 3000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"},
    {"key": "PORT", "value": "3000"}
  ]
}`,
			status: "active",
		},
		{
			id:   3,
			name: "NestJS",
			body: `FROM node:{{ .node_version }}-alpine AS builder

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

CMD ["node", "dist/main"]`,
			config: `{
  "description": "NestJS 엔터프라이즈급 Node.js 프레임워크",
  "categories": ["backend"],
  "display_order": 3,
  "icon_name": "nestjs",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "node_version", "label": "Node.js 버전", "type": "select", "options": ["18", "20", "22"], "default": "20"},
    {"name": "package_manager", "label": "패키지 매니저", "type": "select", "options": ["npm", "yarn", "pnpm"], "default": "npm"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "3000"}
  ],
  "default_ports": [
    {"internal_port": 3000, "network_type": "http"}
  ],
  "default_env": [
    {"key": "NODE_ENV", "value": "production"},
    {"key": "PORT", "value": "3000"}
  ]
}`,
			status: "active",
		},
		{
			id:   4,
			name: "Go Gin",
			body: `FROM golang:{{ .go_version }}-alpine AS builder

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

CMD ["./main"]`,
			config: `{
  "description": "Go Gin 웹 프레임워크",
  "categories": ["backend"],
  "display_order": 4,
  "icon_name": "go",
  "requires_git": true,
  "version": "1.0",
  "template_options": [
    {"name": "go_version", "label": "Go 버전", "type": "select", "options": ["1.21", "1.22", "1.23"], "default": "1.23"},
    {"name": "app_port", "label": "애플리케이션 포트", "type": "number", "options": [], "default": "8080"}
  ],
  "default_ports": [
    {"internal_port": 8080, "network_type": "http"}
  ],
  "default_env": [
    {"key": "GIN_MODE", "value": "release"},
    {"key": "PORT", "value": "8080"}
  ]
}`,
			status: "active",
		},
		{
			id:   5,
			name: "MySQL",
			body: `FROM mysql:{{ .version }}

ENV MYSQL_CHARACTER_SET_SERVER={{ .charset }}

RUN echo "[mysqld]" > /etc/mysql/conf.d/custom.cnf && \
    echo "character-set-server={{ .charset }}" >> /etc/mysql/conf.d/custom.cnf && \
    echo "max_connections={{ .max_connections }}" >> /etc/mysql/conf.d/custom.cnf

EXPOSE 3306

VOLUME ["/var/lib/mysql"]`,
			config: `{
  "description": "MySQL 관계형 데이터베이스",
  "categories": ["database"],
  "display_order": 5,
  "icon_name": "mysql",
  "requires_git": false,
  "version": "1.0",
  "template_options": [
    {"name": "version", "label": "MySQL 버전", "type": "select", "options": ["8.0", "8.4"], "default": "8.4"},
    {"name": "charset", "label": "문자셋", "type": "select", "options": ["utf8mb4", "utf8"], "default": "utf8mb4"},
    {"name": "max_connections", "label": "최대 연결 수", "type": "number", "options": [], "default": "151"}
  ],
  "default_ports": [
    {"internal_port": 3306, "network_type": "tcp"}
  ],
  "default_env": [
    {"key": "MYSQL_ROOT_PASSWORD", "value": ""},
    {"key": "MYSQL_DATABASE", "value": "myapp"},
    {"key": "MYSQL_USER", "value": ""},
    {"key": "MYSQL_PASSWORD", "value": ""}
  ],
  "template_volumes": [
    {"mount_path": "/var/lib/mysql", "capacity": 10240, "description": "MySQL data directory"}
  ]
}`,
			status: "active",
		},
		{
			id:   6,
			name: "PostgreSQL",
			body: `FROM postgres:{{ .version }}

RUN echo "max_connections = {{ .max_connections }}" >> /usr/share/postgresql/postgresql.conf.sample && \
    echo "shared_buffers = {{ .shared_buffers }}" >> /usr/share/postgresql/postgresql.conf.sample

EXPOSE 5432

VOLUME ["/var/lib/postgresql/data"]`,
			config: `{
  "description": "PostgreSQL 관계형 데이터베이스",
  "categories": ["database"],
  "display_order": 6,
  "icon_name": "postgresql",
  "requires_git": false,
  "version": "1.0",
  "template_options": [
    {"name": "version", "label": "PostgreSQL 버전", "type": "select", "options": ["14", "15", "16"], "default": "16"},
    {"name": "max_connections", "label": "최대 연결 수", "type": "number", "options": [], "default": "100"},
    {"name": "shared_buffers", "label": "공유 버퍼 (MB)", "type": "text", "options": [], "default": "128MB"}
  ],
  "default_ports": [
    {"internal_port": 5432, "network_type": "tcp"}
  ],
  "default_env": [
    {"key": "POSTGRES_PASSWORD", "value": ""},
    {"key": "POSTGRES_USER", "value": "postgres"},
    {"key": "POSTGRES_DB", "value": "myapp"}
  ],
  "template_volumes": [
    {"mount_path": "/var/lib/postgresql/data", "capacity": 10240, "description": "PostgreSQL data directory"}
  ]
}`,
			status: "active",
		},
	}

	// Prepare the insert statement
	stmt := `INSERT INTO TEMPLATES (template_id, name, template_body, template_config, status)
	         VALUES (?, ?, ?, ?, ?)`

	for _, tmpl := range templates {
		_, err := db.Exec(stmt, tmpl.id, tmpl.name, tmpl.body, tmpl.config, tmpl.status)
		if err != nil {
			return fmt.Errorf("failed to seed template %s: %w", tmpl.name, err)
		}
	}

	return nil
}

// SeedSystemSettings inserts system settings data for testing
func SeedSystemSettings(db *sql.DB) error {
	// System settings extracted from migration 000001_initial_schema.up.sql
	settings := []struct {
		key         string
		value       string
		valueType   string
		category    string
		description string
		isEditable  bool
	}{
		// Plan base prices
		{"free_plan_base_price", "0", "int", "pricing", "Free plan monthly base price (KRW)", false},
		{"eco_plan_base_price", "1100", "int", "pricing", "Eco plan monthly base price (KRW)", true},
		{"pro_plan_base_price", "14900", "int", "pricing", "Pro plan monthly base price (KRW)", true},

		// Runtime pricing
		{"free_plan_free_minutes", "-1", "int", "pricing", "Free plan free runtime minutes per month (-1 = unlimited)", false},
		{"free_plan_runtime_price_per_minute", "0", "float", "pricing", "Free plan runtime price per minute (KRW)", false},
		{"eco_plan_free_minutes", "500", "int", "pricing", "Eco plan free runtime minutes per month", true},
		{"eco_plan_runtime_price_per_minute", "3.3", "float", "pricing", "Eco plan runtime price per minute (KRW)", true},
		{"pro_plan_free_minutes", "-1", "int", "pricing", "Pro plan free runtime minutes per month (-1 = unlimited)", false},
		{"pro_plan_runtime_price_per_minute", "0", "float", "pricing", "Pro plan runtime price per minute (KRW)", false},

		// Eco plan resource pricing
		{"eco_cpu_price_per_core_per_minute", "2.2", "float", "pricing", "Eco CPU pricing per core per minute (KRW)", true},
		{"eco_memory_price_per_gb_per_minute", "1.1", "float", "pricing", "Eco memory pricing per GB per minute (KRW)", true},
		{"eco_disk_price_per_gb_per_month", "1000", "int", "pricing", "Eco disk pricing per GB per month (KRW)", true},

		// Pro plan resource pricing
		{"pro_cpu_price_per_core_per_month", "5000", "int", "pricing", "Pro CPU pricing per core per month (KRW)", true},
		{"pro_memory_price_per_gb_per_month", "3000", "int", "pricing", "Pro memory pricing per GB per month (KRW)", true},
		{"pro_disk_price_per_gb_per_month", "1000", "int", "pricing", "Pro disk pricing per GB per month (KRW)", true},

		// Free plan limits
		{"free_plan_cpu_limit", "500", "int", "limits", "Free plan fixed CPU limit (millicores)", false},
		{"free_plan_memory_limit", "1024", "int", "limits", "Free plan fixed memory limit (Mi)", false},
		{"free_plan_disk_limit", "1024", "int", "limits", "Free plan fixed disk limit (Mi)", false},
		{"free_plan_max_projects", "1", "int", "limits", "Maximum projects per user for Free plan", true},

		// Beta tier limits
		{"beta_tier_enabled", "true", "boolean", "beta", "Enable beta tier resource restrictions", true},
		{"beta_tier_cpu_limit", "1000", "int", "beta", "Beta tier maximum CPU limit (millicores)", true},
		{"beta_tier_memory_limit", "2048", "int", "beta", "Beta tier maximum memory limit (Mi)", true},
		{"beta_tier_disk_limit", "3072", "int", "beta", "Beta tier maximum disk limit (Mi)", true},

		// Note: max_projects_per_user is now managed in the consolidated 000001_initial_schema.up.sql
	}

	// Prepare the insert statement
	stmt := `INSERT INTO SYSTEM_SETTINGS (setting_key, setting_value, value_type, category, description, is_editable)
	         VALUES (?, ?, ?, ?, ?, ?)
	         ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`

	for _, setting := range settings {
		_, err := db.Exec(stmt, setting.key, setting.value, setting.valueType, setting.category, setting.description, setting.isEditable)
		if err != nil {
			return fmt.Errorf("failed to seed system setting %s: %w", setting.key, err)
		}
	}

	return nil
}
