# Karden — Kubernetes Native Privileged Access Manager

## Project Overview

Kubernetes 네이티브 특권 계정 관리(PAM) 플랫폼.
개발자가 어노테이션만 달면 Janusd가 비밀값을 자동 생성·주입·로테이션·감사한다.

---

## 핵심 설계 원칙

- **K8s Secret이 SSoT(Single Source of Truth)** — 패스워드 값은 etcd(K8s Secret)에 저장, PostgreSQL에는 메타데이터만
- **컨테이너 불변성 존중** — 리눅스 계정 패스워드 변경 없음, exec로 값 변경 없음
- **K8s 표준 방식 준수** — envFrom/secretRef는 개발자가 직접 작성, Janusd가 K8s 표준을 해치지 않음
- **블랙박스 없음** — 모든 레이어 내부 동작 이해하고 구현
- **인터페이스 기반 확장** — SecretStore, Rotator 인터페이스로 Vault 등 연동 대비

---

## 아키텍처

```text
개발자 (어노테이션 부착)
        │
        ▼
Watcher (Controller)
├── K8s Informer로 Pod 어노테이션 실시간 감지 (폴링 아님)
├── 스케일아웃 Pod 인벤토리 자동 관리
└── Janusd Server 트리거

Janusd Server
├── 패스워드 생성 / 로테이션
├── K8s Secret 생성 및 업데이트
├── DB 드라이버 내장 (TCP 직접 접속, exec 아님)
├── Pod Rolling Restart 트리거
├── 로테이션 스케줄러
└── 감사 로그

Web UI
├── 관리 대상 인벤토리 조회
├── 수동 로테이션
├── 수동 비밀값 등록 (외부 API Key 등)
└── 감사 로그 조회

저장소
├── etcd (K8s Secret) — 실제 패스워드 값
└── PostgreSQL — 메타데이터, 감사 로그, 로테이션 이력
```

---

## 어노테이션 스펙

### DB Pod — 어노테이션 상세히

```yaml
annotations:
  janusd.io/inject: "true" # Janusd 관리 대상
  janusd.io/type: "database" # database | secret | manual
  janusd.io/secret-name: "mysql-secret" # Janusd가 생성할 Secret 이름
  janusd.io/db-type: "mysql" # mysql | postgres | mongodb
  janusd.io/db-host: "mysql-svc.default.svc.cluster.local" # DB 호스트
  janusd.io/db-port: "3306" # DB 포트 (기본값: 종류별 자동)
  janusd.io/db-user: "app_user" # 앱이 쓰는 DB 계정 (변경 대상)
  janusd.io/rotation-days: "30" # 로테이션 주기 (기본값: 30)
```

Janusd가 생성하는 Secret 내용 (MySQL 기준)

```yaml
data:
  MYSQL_ROOT_PASSWORD: <자동 생성>
  MYSQL_USER: app_user
  MYSQL_PASSWORD: <자동 생성>
  MYSQL_DATABASE: <janusd.io/db-name 어노테이션 값>
```

### DB Pod 시작 순서 (타이밍 보장)

```text
1. 개발자가 Deployment 배포
   └── envFrom.secretRef 달아둠
   └── Secret 없으므로 Pod Pending 상태로 대기 (K8s 기본 동작)

2. Watcher가 어노테이션 감지
   └── Janusd Server 트리거

3. Janusd Server가 Secret 자동 생성
   └── MYSQL_ROOT_PASSWORD, MYSQL_PASSWORD 등 전부 생성

4. K8s가 Secret 생성 감지
   └── Pod 자동 시작 (별도 트리거 불필요)

5. MySQL이 환경변수 읽어서 초기화
   └── root 패스워드 설정
   └── app_user 계정 생성
```

K8s 기본 동작(Secret 없으면 Pod Pending) 활용 — 별도 트리거 로직 없음

### 앱 Pod — K8s 표준 방식

```yaml
spec:
  containers:
    - name: my-app
      envFrom:
        - secretRef:
            name: mysql-secret # Janusd가 생성한 Secret 참조
```

앱 Pod는 어노테이션 없음. K8s 표준 envFrom 그대로 사용.
Mutating Webhook으로 envFrom 자동 주입은 하지 않음 — K8s 표준 준수, Janusd 종속성 방지.

---

## 비밀값 유형 (janusd.io/type)

| 유형     | 동작                                                         | 예시                            |
| -------- | ------------------------------------------------------------ | ------------------------------- |
| database | 패스워드 생성 → DB ALTER USER → Secret 업데이트 → Pod 재시작 | MySQL, PostgreSQL, MongoDB 계정 |
| secret   | 랜덤값 생성 → Secret 업데이트 → Pod 재시작                   | JWT Secret, 암호화 키           |
| manual   | 웹 UI에서 관리자가 값 직접 입력 → Secret 주입                | 외부 API Key (Stripe 등)        |

---

## DB 로테이션 플로우

```text
1. 새 패스워드 생성
2. DB에 TCP 직접 접속 (Janusd 내장 드라이버, exec 아님)
   ALTER USER 'app_user'@'%' IDENTIFIED BY 'newpass'
3. 성공 → K8s Secret 업데이트
4. 참조 Pod Rolling Restart (무중단)

실패 시 → 롤백 (Secret 변경 없음) + 감사 로그 기록
```

**왜 exec 안 쓰냐**

- distroless 이미지엔 mysql client 없을 수 있음
- DB 레이어 인증(패스워드)과 K8s 레이어 인증(RBAC) 분리
- TCP 직접 접속이 안정적

---

## 컴포넌트별 역할

### Watcher

- K8s Informer 기반 (List-Watch, DeltaFIFO 큐)
- `janusd.io/inject: true` Pod 감지 → PostgreSQL 인벤토리 upsert
- 스케일아웃 Pod 자동 등록, 종료 Pod inactive 처리
- **패스워드 변경 로직 없음** — 인벤토리 관리만

### Janusd Server

- 로테이션 로직 담당
- DB 드라이버 내장 (go-sql-driver/mysql, lib/pq, mongo-driver)
- SecretStore 인터페이스로 저장소 추상화
- Rotator 인터페이스로 유형별 로테이션 추상화

---

## 핵심 인터페이스

```go
// 저장소 추상화 — K8s Secret, Vault, AWS Secrets Manager 교체 가능
type SecretStore interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, value string) error
    Delete(ctx context.Context, key string) error
}

// 로테이션 추상화 — DB 종류, 비밀값 유형별 구현체
type Rotator interface {
    Rotate(ctx context.Context, target Target) error
}
```

---

## PostgreSQL 스키마 (메타데이터)

```sql
-- 관리 대상 인벤토리
managed_targets
├── id
├── pod_name
├── namespace
├── secret_name
├── password_key
├── type (database | secret | manual)
├── db_type (mysql | postgres | mongodb | null)
├── db_host
├── db_port
├── db_user
├── rotation_days
├── last_rotated_at
├── status (active | inactive)
└── created_at

-- 감사 로그
audit_logs
├── id
├── target_id
├── action (rotate | view | manual_set)
├── actor
├── result (success | failure)
├── reason (실패 사유)
├── rotated_at
└── metadata (JSON)
```

---

## 기술 스택

| 영역        | 기술                                      |
| ----------- | ----------------------------------------- |
| 언어        | Go                                        |
| K8s 연동    | client-go, K8s Informer                   |
| DB 드라이버 | go-sql-driver/mysql, lib/pq, mongo-driver |
| 메타 저장소 | PostgreSQL                                |
| 암호화      | AES-256-GCM (Secret 저장 시)              |
| 프론트엔드  | React                                     |
| 배포        | Helm Chart                                |
| 확장 저장소 | Vault (SecretStore 구현체 추가)           |

---

## 디렉토리 구조

```text
janusd/
├── cmd/
│   ├── watcher/          # Watcher 엔트리포인트
│   └── server/           # Janusd Server 엔트리포인트
├── internal/
│   ├── watcher/          # Informer, 인벤토리 관리
│   ├── rotator/          # Rotator 인터페이스 및 구현체
│   │   ├── database/     # MySQL, PostgreSQL, MongoDB
│   │   └── secret/       # 랜덤값 생성
│   ├── store/            # SecretStore 인터페이스 및 구현체
│   │   ├── k8s/          # K8s Secret
│   │   └── vault/        # Vault (추후)
│   ├── scheduler/        # 로테이션 스케줄러
│   ├── audit/            # 감사 로그
│   └── model/            # 공통 도메인 모델
├── api/                  # REST API (Web UI용)
├── web/                  # React Web UI
├── deploy/
│   └── helm/             # Helm Chart
├── CLAUDE.md
└── README.md
```

---

## 주의사항

- `janusd.io/db-user` 는 앱이 쓰는 계정 (변경 대상), Janusd 관리자 계정 아님
- PAM 내부 관리자 계정(pam_admin)은 Janusd가 최초 Secret 생성 시 DB에 자동 생성
  - MySQL root로 첫 접속 → pam_admin 생성 → 이후 root 안 씀
  - root 패스워드도 Janusd가 생성 (MYSQL_ROOT_PASSWORD Secret에 포함)
- 감사 로그는 실패 포함 전부 기록 — 성공만 기록하지 않음
- 리눅스 계정 패스워드는 관리 대상 아님 — 컨테이너 재시작 시 초기화되므로 의미없음
- exec로 컨테이너 내부 접근해서 값 변경하지 않음 — TCP 직접 접속으로 DB 변경
- Mutating Webhook 미사용 — envFrom은 개발자가 직접 작성 (K8s 표준 준수) 추후 도입 고민.
