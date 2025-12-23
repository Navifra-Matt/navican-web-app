# CAN Database Bridge & API Server

SocketCAN 인터페이스에서 CAN 패킷을 읽어 ClickHouse 데이터베이스로 전송하고, REST API로 데이터를 조회할 수 있는 Go 프로젝트입니다.

## 프로젝트 구조

```
/workspace/backend/
├── cmd/                       # 실행 파일 진입점
│   ├── can-reader/           # CAN → Database 브릿지
│   │   └── main.go
│   └── api-server/           # REST API 서버
│       └── main.go
├── internal/                  # 내부 패키지
│   ├── models/               # 공통 데이터 타입
│   │   ├── can.go
│   │   └── query.go
│   ├── can/                  # CAN 리더
│   │   └── reader.go
│   ├── database/             # 데이터베이스 Writer
│   │   ├── writer.go         # Writer 인터페이스
│   │   ├── clickhouse/
│   │   │   ├── config.go
│   │   │   └── writer.go
│   └── api/                  # HTTP API 핸들러
│       ├── server.go
│       ├── clickhouse.go
│       └── utils.go
├── bin/                      # 빌드된 바이너리
│   ├── can-reader
│   └── api-server
├── go.mod
└── go.sum
```

## 주요 기능

### CAN Reader (Data Ingestion)
- SocketCAN 인터페이스에서 CAN 프레임 실시간 읽기
- CAN ID 필터링 지원
- ClickHouse로 배치 전송 (성능 최적화)
- 타임스탬프 자동 기록
- 우아한 종료 (Ctrl+C로 안전하게 종료)
- SocketCAN 인터페이스 통계 자동 수집 및 저장

### API Server (Data Access)
- ClickHouse 데이터 REST API로 조회
- 시간 범위, CAN ID, 인터페이스별 필터링
- SocketCAN 통계 조회 및 집계
- 커스텀 쿼리 실행 (ClickHouse SQL)
- CORS 지원

## 요구사항

- Linux 시스템 (SocketCAN 지원)
- Go 1.21 이상
- ClickHouse 서버 (포트 9000)
- CAN 인터페이스 (예: can0, vcan0)

## 설치

### 의존성 다운로드
```bash
cd /workspace/backend
go mod download
```

### 빌드

```bash
# 두 바이너리 모두 빌드
go build -o bin/can-reader ./cmd/can-reader
go build -o bin/api-server ./cmd/api-server

# 또는 개별 빌드
make build  # Makefile이 있는 경우
```

## 설정 파일 (.env)

프로젝트는 `.env` 파일을 통한 설정을 지원합니다. 이는 명령줄 옵션보다 간편하고 관리하기 쉽습니다.

### .env 파일 생성

```bash
# .env.example을 복사하여 .env 파일 생성
cp .env.example .env

# 또는 환경별 설정 파일 사용
cp .env.development .env  # 개발 환경
cp .env.production .env   # 프로덕션 환경
```

### .env 파일 예시

```env
# CAN Interface Configuration
CAN_INTERFACE=can0
CAN_FILTERS=
STATS_INTERVAL=10

# ClickHouse Configuration
CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=9000
CLICKHOUSE_DATABASE=default
CLICKHOUSE_USERNAME=default
CLICKHOUSE_PASSWORD=
CLICKHOUSE_TABLE=can_messages
CLICKHOUSE_STATS_TABLE=can_interface_stats

# General Configuration
BATCH_SIZE=1000
API_PORT=8080
```

### 설정 옵션 설명

| 옵션 | 설명 | 기본값 |
|------|------|--------|
| `CAN_INTERFACE` | CAN 인터페이스 이름 (예: can0, vcan0) | vcan0 |
| `CAN_FILTERS` | 필터링할 CAN ID (쉼표로 구분, 16진수) | - |
| `STATS_INTERVAL` | 통계 수집 간격 (초) | 10 |
| `CLICKHOUSE_HOST` | ClickHouse 서버 주소 | localhost |
| `CLICKHOUSE_PORT` | ClickHouse 포트 | 9000 |
| `CLICKHOUSE_DATABASE` | ClickHouse 데이터베이스 이름 | default |
| `CLICKHOUSE_USERNAME` | ClickHouse 사용자명 | default |
| `CLICKHOUSE_PASSWORD` | ClickHouse 비밀번호 | - |
| `CLICKHOUSE_TABLE` | CAN 메시지 테이블 이름 | can_messages |
| `CLICKHOUSE_STATS_TABLE` | 통계 테이블 이름 | can_interface_stats |
| `BATCH_SIZE` | 데이터베이스 배치 크기 | 1000 |
| `API_PORT` | API 서버 포트 | 8080 |

---

## 1. CAN Reader 사용법

### 기본 사용 (.env 파일 사용)

```bash
# .env 파일에서 설정 로드
./bin/can-reader

# 다른 경로의 설정 파일 사용
./bin/can-reader -env /path/to/custom.env
```

### 사용 예제

#### 1. 개발 환경에서 실행
```bash
# .env.development 설정 파일 사용
./bin/can-reader -env .env.development
```

#### 2. 프로덕션 환경에서 실행
```bash
# .env.production 설정 파일 사용
./bin/can-reader -env .env.production
```

#### 3. 특정 CAN ID 필터링 (.env 설정)
.env 파일에서 다음과 같이 설정:
```env
CAN_INTERFACE=can0
CAN_FILTERS=100,200,1FF
```

#### 4. SocketCAN 통계 수집
프로그램은 자동으로 `ip -details -statistics link show` 명령어를 실행하여 SocketCAN 인터페이스의 통계를 수집하고 데이터베이스에 저장합니다.

.env 파일에서 수집 간격 설정:
```env
STATS_INTERVAL=10  # 10초마다 통계 수집
```

---

## 2. API Server 사용법

### 서버 실행 (.env 파일 사용)

```bash
# .env 파일에서 설정 로드
./bin/api-server

# 다른 경로의 설정 파일 사용
./bin/api-server -env /path/to/custom.env
```

API 서버는 .env 파일에서 다음 설정을 읽습니다:
- `API_PORT`: API 서버 포트
- `CLICKHOUSE_*`: ClickHouse 연결 정보

### API 엔드포인트 개요

서버 시작 후 다음 URL로 접속하여 전체 API 문서를 확인할 수 있습니다:
```bash
http://localhost:8080/
```

헬스 체크:
```bash
curl http://localhost:8080/health
```

---

## API 엔드포인트 상세

### ClickHouse API

#### 1. 메시지 조회
```bash
GET /api/clickhouse/messages
```

**쿼리 파라미터:**
- `start_time`: 시작 시간 (RFC3339 형식, 예: 2024-01-01T00:00:00Z)
- `end_time`: 종료 시간 (RFC3339 형식)
- `can_id`: CAN ID (10진수 또는 0x로 시작하는 16진수)
- `interface`: CAN 인터페이스 이름 (예: can0, vcan0)
- `limit`: 최대 결과 수 (기본값: 100)
- `offset`: 오프셋

**예제:**
```bash
# 최근 100개 메시지 조회
curl "http://localhost:8080/api/clickhouse/messages?limit=100"

# 특정 CAN ID의 메시지 조회
curl "http://localhost:8080/api/clickhouse/messages?can_id=0x123&limit=50"

# 시간 범위로 조회
curl "http://localhost:8080/api/clickhouse/messages?start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z"
```

**응답 예제:**
```json
[
  {
    "timestamp": "2024-01-01T12:00:00.123456Z",
    "interface": "can0",
    "can_id": 291,
    "can_id_hex": "0x123",
    "dlc": 8,
    "data": [1, 2, 3, 4, 5, 6, 7, 8],
    "data_hex": "01 02 03 04 05 06 07 08"
  }
]
```

#### 2. 메시지 개수 조회
```bash
GET /api/clickhouse/count
curl "http://localhost:8080/api/clickhouse/count?can_id=0x123"
```

#### 3. 고유 CAN ID 목록 조회
```bash
GET /api/clickhouse/can_ids
curl "http://localhost:8080/api/clickhouse/can_ids"
```

#### 4. CAN ID별 통계 조회
```bash
GET /api/clickhouse/stats
curl "http://localhost:8080/api/clickhouse/stats?limit=10"
```

### SocketCAN 통계 API

#### 1. 최신 통계 조회
```bash
GET /api/stats/latest?interface=can0
```

**예제:**
```bash
# 특정 인터페이스의 최신 통계
curl "http://localhost:8080/api/stats/latest?interface=can0"
```

**응답 예제:**
```json
{
  "interface": "can0",
  "timestamp": "2024-01-01T12:00:00Z",
  "state": "UP",
  "bitrate": 500000,
  "bus_state": "ERROR-ACTIVE",
  "rx_packets": 1234567,
  "tx_packets": 987654,
  "rx_errors": 0,
  "tx_errors": 0,
  "rx_error_counter": 0,
  "tx_error_counter": 0
}
```

#### 2. 통계 히스토리 조회
```bash
GET /api/stats/history?interface=can0&start_time=...&limit=100
```

**예제:**
```bash
curl "http://localhost:8080/api/stats/history?interface=can0&limit=100"
```

#### 3. 집계된 통계 조회
시간 간격별로 집계된 통계를 조회합니다.

```bash
GET /api/stats/aggregated?interface=can0&interval=1h
```

**쿼리 파라미터:**
- `interval`: 집계 간격 (1m, 5m, 15m, 1h, 1d)
- `start_time`, `end_time`: 시간 범위
- `interface`: CAN 인터페이스

**예제:**
```bash
# 1시간 간격으로 집계
curl "http://localhost:8080/api/stats/aggregated?interface=can0&interval=1h&limit=24"
```

**응답 예제:**
```json
[
  {
    "time_bucket": "2024-01-01T12:00:00Z",
    "interface": "can0",
    "avg_rx_packets": 1234567.5,
    "avg_tx_packets": 987654.3,
    "total_rx_errors": 0,
    "total_tx_errors": 0,
    "max_bus_error_counter": 0
  }
]
```

## Docker Compose로 실행

프로젝트에 포함된 docker-compose.yml로 ClickHouse를 쉽게 실행할 수 있습니다:

```bash
cd .devcontainer
docker-compose up -d
```

이렇게 하면 다음 서비스가 시작됩니다:
- **ClickHouse**: 포트 9000 (native), 8123 (HTTP)
- **HyperDX**: 포트 8080 (모니터링 UI)

---

## 데이터베이스 스키마

### ClickHouse 테이블

프로그램은 자동으로 다음 스키마의 테이블을 생성합니다:

```sql
CREATE TABLE IF NOT EXISTS can_messages (
    timestamp DateTime64(6),
    interface String,
    can_id UInt32,
    data_0 UInt8,
    data_1 UInt8,
    data_2 UInt8,
    data_3 UInt8,
    data_4 UInt8,
    data_5 UInt8,
    data_6 UInt8,
    data_7 UInt8
) ENGINE = MergeTree()
ORDER BY (timestamp, can_id)
PARTITION BY toYYYYMMDD(timestamp)
SETTINGS index_granularity = 8192
```

---

## 데이터 조회 예제

### ClickHouse SQL 쿼리

#### 최근 데이터 조회
```sql
SELECT * FROM can_messages
ORDER BY timestamp DESC
LIMIT 100;
```

#### 특정 CAN ID 통계
```sql
SELECT
    can_id,
    count() as message_count,
    min(timestamp) as first_seen,
    max(timestamp) as last_seen
FROM can_messages
WHERE timestamp >= now() - INTERVAL 1 HOUR
GROUP BY can_id
ORDER BY message_count DESC;
```

#### 시간대별 메시지 수
```sql
SELECT
    toStartOfMinute(timestamp) as minute,
    count() as messages
FROM can_messages
WHERE timestamp >= now() - INTERVAL 1 HOUR
GROUP BY minute
ORDER BY minute;
```

---

## 가상 CAN 인터페이스로 테스트

실제 CAN 하드웨어가 없는 경우 가상 CAN 인터페이스를 사용할 수 있습니다:

```bash
# vcan0 인터페이스 생성
sudo modprobe vcan
sudo ip link add dev vcan0 type vcan
sudo ip link set up vcan0

# CAN Reader 실행
./bin/can-reader -can vcan0

# 다른 터미널에서 테스트 메시지 전송
cansend vcan0 123#DEADBEEF
cansend vcan0 456#1122334455667788

# API로 데이터 확인
curl "http://localhost:8080/api/clickhouse/messages?limit=10"
```

---

## 성능 최적화

### 배치 크기 조정
- **배치 크기**: `-batch` 옵션으로 조정 (기본값: 1000)
  - 높은 값: 처리량 증가, 메모리 사용 증가
  - 낮은 값: 실시간성 향상, CPU 사용 증가

### 파티셔닝
- 테이블은 일별로 자동 파티셔닝됩니다
  - 오래된 데이터 삭제가 용이
  - 쿼리 성능 향상

---

## 모니터링

### CAN Reader 통계

프로그램은 1000개 메시지마다 통계를 출력합니다:

```
2025/11/24 12:00:00 Processed 1000 messages (errors: 0)
2025/11/24 12:00:05 Flushed 1000 messages to ClickHouse
```

### API Server 로그

```
2025/11/24 12:00:00 Starting API server on :8080
2025/11/24 12:00:01 [GET] /api/clickhouse/messages 192.168.1.100:12345
2025/11/24 12:00:01 [GET] /api/clickhouse/messages completed in 15ms
```

---


## 개발 팁

### 1. jq를 사용한 JSON 포맷팅
```bash
curl "http://localhost:8080/api/clickhouse/messages?limit=10" | jq '.'
```

### 2. 시간 필터링 예제
```bash
# 최근 1시간 데이터 조회
START_TIME=$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)
END_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
curl "http://localhost:8080/api/clickhouse/messages?start_time=${START_TIME}&end_time=${END_TIME}"
```

### 3. Python으로 API 호출
```python
import requests
from datetime import datetime, timedelta

# API 엔드포인트
base_url = "http://localhost:8080"

# 최근 1시간 데이터 조회
end_time = datetime.utcnow()
start_time = end_time - timedelta(hours=1)

params = {
    "start_time": start_time.strftime("%Y-%m-%dT%H:%M:%SZ"),
    "end_time": end_time.strftime("%Y-%m-%dT%H:%M:%SZ"),
    "can_id": "0x123",
    "limit": 100
}

response = requests.get(f"{base_url}/api/clickhouse/messages", params=params)
data = response.json()

print(f"Found {len(data)} messages")
for msg in data:
    print(f"CAN ID: {msg['can_id_hex']}, Data: {msg['data_hex']}")
```

### 4. JavaScript/Node.js로 API 호출
```javascript
const axios = require('axios');

const baseURL = 'http://localhost:8080';

async function getCANMessages() {
  try {
    const response = await axios.get(`${baseURL}/api/clickhouse/messages`, {
      params: {
        can_id: '0x123',
        limit: 100
      }
    });

    console.log(`Found ${response.data.length} messages`);
    response.data.forEach(msg => {
      console.log(`CAN ID: ${msg.can_id_hex}, Data: ${msg.data_hex}`);
    });
  } catch (error) {
    console.error('Error:', error.message);
  }
}

getCANMessages();
```

---

## 트러블슈팅

### CAN 인터페이스를 찾을 수 없음

```bash
# CAN 인터페이스 확인
ip link show

# CAN 인터페이스 활성화
sudo ip link set can0 up type can bitrate 500000
```

### ClickHouse 연결 실패

- ClickHouse 서버가 실행 중인지 확인
- 방화벽 설정 확인 (기본 포트: 9000)
- 사용자 권한 확인

### 권한 오류

일부 시스템에서는 CAN 인터페이스 접근에 root 권한이 필요할 수 있습니다:

```bash
sudo ./bin/can-reader -can can0
```

### API 에러 응답

모든 에러는 다음과 같은 형식으로 반환됩니다:

```json
{
  "error": "Error message description"
}
```

**일반적인 HTTP 상태 코드:**
- `200 OK`: 성공
- `400 Bad Request`: 잘못된 요청 파라미터
- `404 Not Found`: 리소스를 찾을 수 없음
- `500 Internal Server Error`: 서버 내부 오류
- `503 Service Unavailable`: 데이터베이스 연결 불가

---

## 라이선스

MIT License

## 기여

버그 리포트와 개선 제안을 환영합니다!
