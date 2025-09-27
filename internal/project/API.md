# Project Bounded Context API 명세

## API 개요

Project Bounded Context는 컨테이너 플랫폼의 프로젝트 및 볼륨 관리를 담당하는 API입니다.

- **Base URL**: `https://api.launchpad.kr`
- **API Version**: v1
- **Content-Type**: `application/json`
- **Authentication**: Bearer Token (JWT)

## 인증

모든 API는 Bearer Token을 통한 인증이 필요합니다.

```http
Authorization: Bearer <JWT_TOKEN>
```

## 공통 응답 형식

### 성공 응답
```json
{
  "status": "success",
  "data": {
    // 실제 데이터
  }
}
```

### 에러 응답
```json
{
  "status": "error",
  "error": {
    "code": "ERROR_CODE",
    "message": "Error description"
  }
}
```

## 에러 코드 정의

### 프로젝트 관련 에러
| 에러 코드 | HTTP Status | 설명 |
|-----------|-------------|------|
| `PROJECT_NOT_FOUND` | 404 | 프로젝트를 찾을 수 없음 |
| `PROJECT_ALREADY_EXISTS` | 409 | 프로젝트가 이미 존재함 |
| `SLUG_ALREADY_EXISTS` | 409 | 슬러그가 이미 존재함 |
| `INVALID_PROJECT_DATA` | 400 | 유효하지 않은 프로젝트 데이터 |
| `PROJECT_NOT_ACTIVE` | 403 | 프로젝트가 활성 상태가 아님 |
| `CANNOT_MODIFY_DELETED_PROJECT` | 403 | 삭제된 프로젝트는 수정할 수 없음 |

### 권한 관련 에러
| 에러 코드 | HTTP Status | 설명 |
|-----------|-------------|------|
| `PERMISSION_DENIED` | 403 | 권한 거부 |
| `OWNER_REQUIRED` | 403 | 소유자 권한 필요 |
| `USER_ALREADY_IN_PROJECT` | 409 | 사용자가 이미 프로젝트에 참여 |
| `USER_NOT_IN_PROJECT` | 404 | 사용자가 프로젝트에 참여하지 않음 |

### 유효성 검사 에러
| 에러 코드 | HTTP Status | 설명 |
|-----------|-------------|------|
| `NAME_REQUIRED` | 400 | 프로젝트 이름 필수 |
| `SLUG_REQUIRED` | 400 | 슬러그 필수 |
| `INVALID_SLUG` | 400 | 유효하지 않은 슬러그 |
| `INVALID_PROJECT_ID` | 400 | 유효하지 않은 프로젝트 ID |
| `VALIDATION_FAILED` | 400 | 유효성 검사 실패 |

### 리소스 관련 에러
| 에러 코드 | HTTP Status | 설명 |
|-----------|-------------|------|
| `RESOURCE_LIMIT_EXCEEDED` | 403 | 리소스 제한 초과 |
| `PLAN_LIMIT_EXCEEDED` | 403 | 플랜 제한 초과 |

### 볼륨 관련 에러
| 에러 코드 | HTTP Status | 설명 |
|-----------|-------------|------|
| `VOLUME_NOT_FOUND` | 404 | 볼륨을 찾을 수 없음 |
| `VOLUME_NAME_REQUIRED` | 400 | 볼륨 이름 필수 |
| `INVALID_CAPACITY` | 400 | 유효하지 않은 볼륨 용량 |
| `DUPLICATE_VOLUME_NAME` | 409 | 중복된 볼륨 이름 |

### 인프라 에러
| 에러 코드 | HTTP Status | 설명 |
|-----------|-------------|------|
| `DATABASE_UNAVAILABLE` | 503 | 데이터베이스 일시적 사용 불가 |
| `DATABASE_OPERATION_FAILED` | 500 | 데이터베이스 작업 실패 |

## 데이터 타입 정의

### ProjectStatus
- `active`: 활성 상태
- `inactive`: 비활성 상태
- `suspended`: 일시 중단 상태
- `deleted`: 삭제 상태

### ProjectUserRole
- `owner`: 소유자
- `admin`: 관리자
- `member`: 멤버

### 리소스 제한
- **CPU Limit**: 0-4000 millicores (1000 = 1 CPU)
- **Memory Request/Limit**: 128-8192 Mi (Mebibytes)
- **Disk Limit**: 128-10240 Mi (Mebibytes)
- **Traffic Limit**: 최소 128 Mi, 상한 없음

### 볼륨 용량
- **Capacity**: 128-10240 Mi (Mebibytes)

---

# 프로젝트 API

## 1. 프로젝트 생성

**POST** `/api/v1/projects`

새로운 프로젝트를 생성합니다.

### 요청

#### Headers
```http
Content-Type: application/json
Authorization: Bearer <token>
```

#### Body
```json
{
  "name": "My Project",
  "slug": "myproject",
  "fqdn": "myproject.example.com",
  "plan": "basic",
  "cpu_limit": 1000,
  "memory_request": 512,
  "memory_limit": 1024,
  "disk_limit": 2048,
  "traffic_limit": 10240
}
```

#### 필드 설명
| 필드 | 타입 | 필수 | 제약사항 | 설명 |
|------|------|------|----------|------|
| `name` | string | ✓ | 1-100자 | 프로젝트 이름 |
| `slug` | string | ✓ | 3-63자, 영숫자만 | URL 슬러그 |
| `fqdn` | string | ✗ | 최대 253자, FQDN 형식 | 도메인 이름 |
| `plan` | string | ✗ | - | 플랜 (미구현) |
| `cpu_limit` | uint32 | ✗ | 0-4000 | CPU 제한 (밀리코어) |
| `memory_request` | uint32 | ✗ | 128-8192 | 메모리 요청 (Mi) |
| `memory_limit` | uint32 | ✗ | 128-8192 | 메모리 제한 (Mi) |
| `disk_limit` | uint32 | ✗ | 128-10240 | 디스크 제한 (Mi) |
| `traffic_limit` | uint64 | ✗ | 최소 128 | 트래픽 제한 (Mi) |

### 응답

#### 성공 (201 Created)
```json
{
  "status": "success",
  "data": {
    "project_id": 1,
    "name": "My Project",
    "slug": "myproject",
    "fqdn": "myproject.example.com",
    "plan": "basic",
    "status": "active",
    "created_at": "2023-01-01T00:00:00Z"
  }
}
```

#### 실패 예시
```json
{
  "status": "error",
  "error": {
    "code": "SLUG_ALREADY_EXISTS",
    "message": "Slug already exists"
  }
}
```

---

## 2. 프로젝트 목록 조회

**GET** `/api/v1/projects`

사용자가 접근 가능한 프로젝트 목록을 조회합니다.

### 요청

#### Headers
```http
Authorization: Bearer <token>
```

#### Query Parameters
| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| `user_id` | uint | ✗ | 현재 사용자 | 특정 사용자의 프로젝트 조회 |
| `offset` | int | ✗ | 0 | 시작 위치 |
| `limit` | int | ✗ | 10 | 조회 개수 (최대 100) |

### 응답

#### 성공 (200 OK)
```json
{
  "status": "success",
  "data": {
    "projects": [
      {
        "project_id": 1,
        "name": "My Project",
        "slug": "myproject",
        "fqdn": "myproject.example.com",
        "status": "active",
        "created_at": "2023-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
}
```

---

## 3. 프로젝트 상세 조회

**GET** `/api/v1/projects/:id`

특정 프로젝트의 상세 정보를 조회합니다. ID 대신 slug도 사용 가능합니다.

### 요청

#### Headers
```http
Authorization: Bearer <token>
```

#### Path Parameters
| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `id` | uint/string | 프로젝트 ID 또는 slug |

### 응답

#### 성공 (200 OK)
```json
{
  "status": "success",
  "data": {
    "project_id": 1,
    "name": "My Project",
    "slug": "myproject",
    "fqdn": "myproject.example.com",
    "plan": "basic",
    "status": "active",
    "cpu_limit": 1000,
    "memory_limit": 1024,
    "disk_limit": 2048,
    "traffic_limit": 10240,
    "users": [
      {
        "user_id": 1,
        "role": "owner",
        "created_at": "2023-01-01T00:00:00Z"
      }
    ],
    "volumes": [
      {
        "volume_id": 1,
        "name": "data",
        "capacity": 1024,
        "created_at": "2023-01-01T00:00:00Z"
      }
    ],
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z"
  }
}
```

#### 실패 예시
```json
{
  "status": "error",
  "error": {
    "code": "PROJECT_NOT_FOUND",
    "message": "Project not found"
  }
}
```

### 권한
- 프로젝트에 참여한 사용자만 조회 가능
- 권한이 없으면 `PROJECT_NOT_FOUND` 에러 반환 (정보 노출 방지)

---

## 4. 프로젝트 수정

**PUT** `/api/v1/projects/:id`

기존 프로젝트의 정보를 수정합니다.

### 요청

#### Headers
```http
Content-Type: application/json
Authorization: Bearer <token>
```

#### Path Parameters
| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `id` | uint | 프로젝트 ID |

#### Body
```json
{
  "name": "Updated Project Name",
  "fqdn": "updated.example.com",
  "plan": "premium",
  "status": "active",
  "cpu_limit": 2000,
  "memory_request": 1024,
  "memory_limit": 2048,
  "disk_limit": 4096,
  "traffic_limit": 20480
}
```

#### 필드 설명
모든 필드는 선택사항이며, 제공된 필드만 업데이트됩니다.

| 필드 | 타입 | 제약사항 | 설명 |
|------|------|----------|------|
| `name` | string | 1-100자 | 프로젝트 이름 |
| `fqdn` | string | 최대 253자, FQDN 형식 | 도메인 이름 |
| `plan` | string | - | 플랜 (미구현) |
| `status` | string | active/inactive/suspended | 프로젝트 상태 |
| `cpu_limit` | uint32 | 0-4000 | CPU 제한 (밀리코어) |
| `memory_request` | uint32 | 128-8192 | 메모리 요청 (Mi) |
| `memory_limit` | uint32 | 128-8192 | 메모리 제한 (Mi) |
| `disk_limit` | uint32 | 128-10240 | 디스크 제한 (Mi) |
| `traffic_limit` | uint64 | 최소 128 | 트래픽 제한 (Mi) |

### 응답

#### 성공 (200 OK)
```json
{
  "status": "success",
  "data": {
    "project_id": 1,
    "name": "Updated Project Name",
    "slug": "myproject",
    "fqdn": "updated.example.com",
    "plan": "premium",
    "status": "active",
    "updated_at": "2023-01-01T12:00:00Z"
  }
}
```

### 권한
- 프로젝트의 소유자 또는 관리자만 수정 가능
- 권한이 없으면 `PERMISSION_DENIED` 에러 반환

---

## 5. 프로젝트 삭제

**DELETE** `/api/v1/projects/:id`

프로젝트를 삭제합니다.

### 요청

#### Headers
```http
Authorization: Bearer <token>
```

#### Path Parameters
| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `id` | uint | 프로젝트 ID |

### 응답

#### 성공 (200 OK)
```json
{
  "status": "success",
  "data": {
    "message": "Project deleted successfully"
  }
}
```

### 권한
- 프로젝트의 소유자 또는 관리자만 삭제 가능
- 권한이 없으면 `PERMISSION_DENIED` 에러 반환

---

# 볼륨 API

## 1. 볼륨 추가

**POST** `/api/v1/volumes`

프로젝트에 새로운 볼륨을 추가합니다.

### 요청

#### Headers
```http
Content-Type: application/json
Authorization: Bearer <token>
```

#### Body
```json
{
  "project_id": 1,
  "name": "data",
  "capacity": 1024
}
```

#### 필드 설명
| 필드 | 타입 | 필수 | 제약사항 | 설명 |
|------|------|------|----------|------|
| `project_id` | uint | ✓ | - | 프로젝트 ID |
| `name` | string | ✓ | 1-63자, 영숫자만 | 볼륨 이름 |
| `capacity` | uint32 | ✓ | 128-10240 | 볼륨 용량 (Mi) |

### 응답

#### 성공 (201 Created)
```json
{
  "status": "success",
  "data": {
    "volume_id": 1,
    "project_id": 1,
    "name": "data",
    "capacity": 1024,
    "created_at": "2023-01-01T00:00:00Z"
  }
}
```

### 권한
- 해당 프로젝트의 소유자 또는 관리자만 볼륨 추가 가능

---

## 2. 볼륨 목록 조회

**GET** `/api/v1/volumes`

볼륨 목록을 조회합니다.

### 요청

#### Headers
```http
Authorization: Bearer <token>
```

#### Query Parameters
| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `project_id` | uint | ✗ | 특정 프로젝트의 볼륨만 조회 |

### 응답

#### 성공 (200 OK)
```json
{
  "status": "success",
  "data": {
    "volumes": [
      {
        "volume_id": 1,
        "project_id": 1,
        "name": "data",
        "capacity": 1024,
        "created_at": "2023-01-01T00:00:00Z"
      }
    ]
  }
}
```

### 권한
- `project_id` 파라미터가 있는 경우: 해당 프로젝트에 접근 권한이 있는 사용자만 조회 가능
- `project_id` 파라미터가 없는 경우: 사용자가 접근 가능한 모든 프로젝트의 볼륨을 필터링하여 반환

---

## 3. 볼륨 수정

**PUT** `/api/v1/volumes/:id`

기존 볼륨의 정보를 수정합니다.

### 요청

#### Headers
```http
Content-Type: application/json
Authorization: Bearer <token>
```

#### Path Parameters
| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `id` | uint | 볼륨 ID |

#### Body
```json
{
  "name": "updated-data",
  "capacity": 2048
}
```

#### 필드 설명
모든 필드는 선택사항이며, 제공된 필드만 업데이트됩니다.

| 필드 | 타입 | 제약사항 | 설명 |
|------|------|----------|------|
| `name` | string | 1-63자, 영숫자만 | 볼륨 이름 |
| `capacity` | uint32 | 128-10240 | 볼륨 용량 (Mi) |

### 응답

#### 성공 (200 OK)
```json
{
  "status": "success",
  "data": {
    "volume_id": 1,
    "project_id": 1,
    "name": "updated-data",
    "capacity": 2048,
    "updated_at": "2023-01-01T12:00:00Z"
  }
}
```

### 권한
- 볼륨이 속한 프로젝트의 소유자 또는 관리자만 수정 가능
- 권한이 없으면 `VOLUME_NOT_FOUND` 에러 반환 (정보 노출 방지)

---

## 4. 볼륨 삭제

**DELETE** `/api/v1/volumes/:id`

볼륨을 삭제합니다.

### 요청

#### Headers
```http
Authorization: Bearer <token>
```

#### Path Parameters
| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `id` | uint | 볼륨 ID |

### 응답

#### 성공 (200 OK)
```json
{
  "status": "success",
  "data": {
    "message": "Volume removed successfully"
  }
}
```

### 권한
- 볼륨이 속한 프로젝트의 소유자 또는 관리자만 삭제 가능
- 권한이 없으면 `VOLUME_NOT_FOUND` 에러 반환 (정보 노출 방지)

---

## 보안 고려사항

### 권한 체크
- 모든 API는 JWT 토큰을 통한 인증 필요
- 프로젝트별 세분화된 권한 제어
- 정보 노출 방지를 위한 에러 메시지 통일

### 데이터 유효성
- 요청 데이터의 엄격한 유효성 검사
- SQL 인젝션 방지를 위한 파라미터화된 쿼리 사용
- XSS 방지를 위한 출력 인코딩

### 리소스 제한
- API 호출 빈도 제한 (Rate Limiting)
- 페이지네이션을 통한 대용량 데이터 처리
- 리소스 사용량 모니터링 및 제한

---

## 변경 이력

### v1.0.0 (2024-01-01)
- 프로젝트 CRUD API 구현
- 볼륨 관리 API 구현
- JWT 기반 인증 시스템
- 권한 기반 접근 제어
- 리소스 제한 관리