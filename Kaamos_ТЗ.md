# ├── cmd/

│   └── bot/
│       └── main.go

нет, так не надо.  main.go всегда лежит в корне проекта. Обнови ТЗ с учётом правки

Понял, обновляю ТЗ. Для MVP с тремя отдельными компонентами (бот, сервер, фронтенд) используем подход с **отдельными репозиториями** (multirepo), где `main.go` находится в корне каждого проекта.[^1][^2][^3]

***

# Техническое задание: Kaamos MVP

## 1. Telegram Bot (Golang)

### 1.1 Общее описание

Telegram бот для управления доступом к видеозвонкам и генерации ссылок на комнаты.[^4][^5]

### 1.2 Технологический стек

- **Язык:** Golang 1.21+
- **Библиотека:** `github.com/go-telegram-bot-api/telegram-bot-api/v5`
- **Архитектура:** Простая структура для малого проекта
- **База данных (для MVP):** In-memory map для хранения активных сессий


### 1.3 Функциональные требования

#### 1.3.1 Команда /start

**Описание:** Инициализация бота и запрос ключа доступа

**Поведение:**

1. Приветственное сообщение: "Добро пожаловать в Kaamos! Введите ключ доступа:"
2. Сохранение состояния пользователя (ожидание ключа)
3. Следующее сообщение обрабатывается как ключ

**Структура состояния:**

```go
type UserState struct {
    UserID      int64
    State       string // "awaiting_key", "authorized"
    KeyExpiry   time.Time
    CreatedAt   time.Time
}
```


#### 1.3.2 Верификация ключа

**Описание:** Проверка введённого ключа через Key Service

**Логика для MVP:**

- Захардкоженный ключ: `KAAMOS_MVP_2025`
- Срок действия: 6 месяцев с момента первой авторизации
- Хранение в памяти: `map[int64]*UserState`

**Ответ пользователю:**

- ✅ Успех: "Ключ принят! Используйте /call для создания комнаты"
- ❌ Ошибка: "Неверный ключ. Попробуйте снова или обратитесь к администратору"


#### 1.3.3 Команда /call

**Описание:** Генерация ссылки на новую комнату звонка

**Процесс:**

1. Проверка авторизации пользователя
2. POST запрос к Kaamos Server: `POST /api/rooms/create`
3. Получение `room_id` и URL
4. Отправка сообщения пользователю с кликабельной ссылкой

**Формат ответа:**

```
🎥 Ваша комната готова!
Ссылка: https://kaamos.yourdomain.com/room/{room_id}

Перешлите эту ссылку собеседнику для начала звонка
Комната действительна 24 часа
```


#### 1.3.4 Обработка ошибок

- Неавторизованный доступ к /call → "Сначала введите ключ через /start"
- Недоступен Kaamos Server → "Сервис временно недоступен. Попробуйте позже"
- Истёк срок ключа → "Ваш ключ истёк. Обратитесь к администратору"


### 1.4 Нефункциональные требования

- Graceful shutdown с сохранением состояния
- Логирование всех действий (уровни: INFO, ERROR)
- Обработка до 100 одновременных пользователей
- Время отклика < 2 секунд на команду


### 1.5 Структура проекта

```
kaamos-bot/
├── main.go                    # Точка входа приложения
├── go.mod
├── go.sum
├── README.md
├── .env.example
├── config.go                  # Конфигурация
├── handlers.go                # Обработчики команд Telegram
├── auth.go                    # Логика авторизации
├── room_client.go             # HTTP клиент для Kaamos Server
├── storage.go                 # In-memory хранилище состояний
└── models.go                  # Структуры данных
```

**Обоснование структуры:**
Для небольшого проекта с простой логикой используем flat structure в корне. Все файлы в одной директории упрощают навигацию и избавляют от избыточной вложенности.

***

## 2. Kaamos Server (Golang + WebRTC SFU)

### 2.1 Общее описание

WebRTC SFU сервер для обработки видеозвонков с signaling через WebSocket.

### 2.2 Технологический стек

- **Язык:** Golang 1.21+
- **WebRTC:** `github.com/pion/webrtc/v4`
- **WebSocket:** `github.com/gorilla/websocket`
- **HTTP Router:** `github.com/gorilla/mux`
- **Архитектура:** SFU (Selective Forwarding Unit)


### 2.3 API Endpoints

#### 2.3.1 POST /api/rooms/create

**Описание:** Создание новой комнаты для звонка

**Request:**

```json
{
  "creator_user_id": 123456789
}
```

**Response:**

```json
{
  "room_id": "abc123xyz",
  "url": "https://kaamos.yourdomain.com/room/abc123xyz",
  "expires_at": "2025-11-24T14:48:00Z"
}
```


#### 2.3.2 GET /api/rooms/:room_id

**Описание:** Проверка существования комнаты

**Response:**

```json
{
  "room_id": "abc123xyz",
  "active": true,
  "participants": 1,
  "created_at": "2025-11-23T14:48:00Z"
}
```


#### 2.3.3 WebSocket /ws/:room_id

**Описание:** Signaling для WebRTC соединения[^12][^13][^14]

**Протокол сообщений (JSON):**

```json
// Client → Server: Join room
{
  "type": "join",
  "room_id": "abc123xyz",
  "user_id": "user_123"
}

// Client → Server: WebRTC Offer/Answer
{
  "type": "offer",
  "sdp": "v=0...",
  "room_id": "abc123xyz"
}

// Server → Client: ICE candidates
{
  "type": "ice-candidate",
  "candidate": "...",
  "sdpMid": "0"
}
```


### 2.4 WebRTC SFU логика

#### 2.4.1 Управление комнатами

```go
type Room struct {
    ID           string
    Participants map[string]*Peer
    CreatedAt    time.Time
    ExpiresAt    time.Time
    mu           sync.RWMutex
}

type Peer struct {
    ID         string
    Conn       *websocket.Conn
    PeerConn   *webrtc.PeerConnection
    Tracks     []*webrtc.TrackRemote
}
```


#### 2.4.2 Форвардинг медиа

- Получение трека от одного peer → Отправка всем остальным в комнате
- Поддержка VP8, H264 кодеков
- Автоматическая адаптация битрейта


### 2.5 Нефункциональные требования

- Поддержка до 50 одновременных комнат
- Максимум 4 участника на комнату (для MVP)
- Автоудаление комнат через 24 часа неактивности
- STUN сервер: Google STUN (`stun:stun.l.google.com:19302`)
- Graceful shutdown с закрытием всех соединений


### 2.6 Структура проекта

```
kaamos-server/
├── main.go                    # Точка входа, инициализация HTTP сервера
├── go.mod
├── go.sum
├── README.md
├── .env.example
├── config.go                  # Конфигурация
├── handlers.go                # HTTP handlers для API
├── websocket.go               # WebSocket handler и signaling
├── room.go                    # Управление комнатами
├── peer.go                    # Управление peer connections
├── sfu.go                     # SFU логика форвардинга треков
├── models.go                  # Структуры данных
└── static/                    # Статические файлы фронтенда (для деплоя)
    ├── index.html
    ├── style.css
    └── app.js
```

**Обоснование структуры:**
Для MVP с умеренной сложностью используем flat structure с логическим разделением по файлам. Это проще cmd/ подхода и достаточно для одного executable.

***

## 3. Web Frontend (Vanilla JavaScript)

### 3.1 Общее описание

Минималистичный одностраничный интерфейс для видеозвонков.

### 3.2 Технологический стек

- **HTML5** + **CSS3** + **Vanilla JavaScript**
- **WebRTC API** (getUserMedia, RTCPeerConnection)
- **WebSocket API** для signaling
- **Сборщик (опционально):** Vite для production bundle


### 3.3 Файловая структура

```
kaamos-frontend/
├── index.html                 # Главная страница
├── style.css                  # Стили
├── app.js                     # Основная логика и инициализация
├── webrtc.js                  # WebRTC управление
├── signaling.js               # WebSocket взаимодействие
├── package.json               # Только если используешь Vite для сборки
├── vite.config.js             # Конфигурация Vite (опционально)
└── README.md
```

**Примечание:** Для development можно открывать `index.html` напрямую через `python3 -m http.server`. Для production собирать через Vite в `dist/` и выкладывать на Kaamos Server в `static/`.

### 3.4 UI Компоненты

#### 3.4.1 Layout (index.html)

```html
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Kaamos Call</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <div id="app">
        <!-- Статус соединения -->
        <div id="status">Подключение...</div>
        
        <!-- Видео контейнер -->
        <div class="video-container">
            <video id="remoteVideo" autoplay playsinline></video>
            <video id="localVideo" autoplay muted playsinline></video>
        </div>
        
        <!-- Панель управления -->
        <div class="controls">
            <button id="toggleMic" class="control-btn active">
                <span class="icon">🎤</span>
            </button>
            <button id="toggleCam" class="control-btn active">
                <span class="icon">📹</span>
            </button>
            <button id="endCall" class="control-btn danger">
                <span class="icon">📞</span>
                <span class="label">Завершить</span>
            </button>
        </div>
    </div>
    <script type="module" src="app.js"></script>
</body>
</html>
```


#### 3.4.2 Стили (style.css)

**Требования к дизайну:**

- Responsive дизайн (мобильные + десктоп)
- Тёмная тема (минимальная нагрузка на глаза)
- Крупные кнопки для мобильных устройств (минимум 48x48px)
- Плавные анимации переходов состояний
- Picture-in-picture режим для локального видео

**Ключевые элементы:**

```css
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    background: #000;
    overflow: hidden;
}

#app {
    width: 100vw;
    height: 100vh;
    position: relative;
}

#status {
    position: absolute;
    top: 20px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(0,0,0,0.8);
    color: #fff;
    padding: 12px 24px;
    border-radius: 20px;
    font-size: 14px;
    z-index: 10;
    transition: opacity 0.3s;
}

.video-container {
    position: relative;
    width: 100%;
    height: 100vh;
}

#remoteVideo {
    width: 100%;
    height: 100%;
    object-fit: cover;
    background: #1a1a1a;
}

#localVideo {
    position: absolute;
    bottom: 100px;
    right: 20px;
    width: 120px;
    height: 160px;
    border-radius: 12px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.5);
    object-fit: cover;
    background: #2d2d2d;
}

.controls {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 16px;
    padding: 20px;
    background: linear-gradient(to top, rgba(0,0,0,0.9), transparent);
}

.control-btn {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    border: none;
    background: #2d2d2d;
    color: #fff;
    font-size: 24px;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
}

.control-btn:hover {
    transform: scale(1.1);
}

.control-btn:active {
    transform: scale(0.95);
}

.control-btn.active {
    background: #4a9eff;
}

.control-btn.danger {
    background: #ff4444;
    width: auto;
    padding: 0 24px;
    border-radius: 28px;
    gap: 8px;
}

.control-btn .label {
    font-size: 14px;
    font-weight: 600;
}

/* Mobile responsiveness */
@media (max-width: 768px) {
    #localVideo {
        width: 100px;
        height: 133px;
        bottom: 90px;
        right: 12px;
    }
    
    .controls {
        padding: 12px;
        gap: 12px;
    }
    
    .control-btn {
        width: 48px;
        height: 48px;
        font-size: 20px;
    }
}
```


### 3.5 JavaScript логика

#### 3.5.1 app.js — Инициализация

```javascript
import { SignalingClient } from './signaling.js';
import { WebRTCManager } from './webrtc.js';

// Получение room_id из URL
const pathParts = window.location.pathname.split('/');
const roomId = pathParts[pathParts.length - 1];

if (!roomId) {
    alert('Неверная ссылка на комнату');
    throw new Error('No room ID in URL');
}

// Инициализация компонентов
const signaling = new SignalingClient(roomId);
const webrtc = new WebRTCManager(signaling);

// Запуск приложения
async function init() {
    try {
        updateStatus('Запрос доступа к камере...');
        
        // Запрос доступа к камере/микрофону
        const localStream = await navigator.mediaDevices.getUserMedia({
            video: { 
                width: { ideal: 1280 },
                height: { ideal: 720 },
                facingMode: 'user'
            },
            audio: {
                echoCancellation: true,
                noiseSuppression: true,
                autoGainControl: true
            }
        });
        
        // Отображение локального видео
        document.getElementById('localVideo').srcObject = localStream;
        
        updateStatus('Подключение к серверу...');
        
        // Подключение к signaling серверу
        await signaling.connect();
        
        // Настройка WebRTC
        await webrtc.initialize(localStream);
        
        updateStatus('Звонок активен');
    } catch (error) {
        console.error('Initialization error:', error);
        if (error.name === 'NotAllowedError') {
            updateStatus('Доступ к камере/микрофону запрещён');
        } else if (error.name === 'NotFoundError') {
            updateStatus('Камера или микрофон не найдены');
        } else {
            updateStatus('Ошибка соединения');
        }
    }
}

// UI обработчики
document.getElementById('toggleMic').addEventListener('click', (e) => {
    const isEnabled = webrtc.toggleAudio();
    e.currentTarget.classList.toggle('active', isEnabled);
});

document.getElementById('toggleCam').addEventListener('click', (e) => {
    const isEnabled = webrtc.toggleVideo();
    e.currentTarget.classList.toggle('active', isEnabled);
});

document.getElementById('endCall').addEventListener('click', () => {
    webrtc.hangup();
    document.body.innerHTML = '<div style="display:flex;align-items:center;justify-content:center;height:100vh;color:#fff;font-size:20px;">Звонок завершён</div>';
});

function updateStatus(message) {
    const statusEl = document.getElementById('status');
    statusEl.textContent = message;
    statusEl.style.display = 'block';
    
    // Скрыть через 3 секунды если успешное соединение
    if (message === 'Звонок активен') {
        setTimeout(() => {
            statusEl.style.opacity = '0';
        }, 3000);
    }
}

// Запуск
init();
```


#### 3.5.2 webrtc.js — WebRTC управление

```javascript
export class WebRTCManager {
    constructor(signaling) {
        this.signaling = signaling;
        this.peerConnection = null;
        this.localStream = null;
        
        // Регистрация обработчиков signaling сообщений
        this.signaling.on('offer', (data) => this.handleOffer(data));
        this.signaling.on('answer', (data) => this.handleAnswer(data));
        this.signaling.on('ice-candidate', (data) => this.handleIceCandidate(data));
    }
    
    async initialize(localStream) {
        this.localStream = localStream;
        
        // Конфигурация RTCPeerConnection
        const config = {
            iceServers: [
                { urls: 'stun:stun.l.google.com:19302' },
                { urls: 'stun:stun1.l.google.com:19302' }
            ]
        };
        
        this.peerConnection = new RTCPeerConnection(config);
        
        // Добавление локальных треков
        localStream.getTracks().forEach(track => {
            this.peerConnection.addTrack(track, localStream);
        });
        
        // Обработка удалённых треков
        this.peerConnection.ontrack = (event) => {
            console.log('Received remote track');
            const remoteVideo = document.getElementById('remoteVideo');
            if (remoteVideo.srcObject !== event.streams[^0]) {
                remoteVideo.srcObject = event.streams[^0];
            }
        };
        
        // Обработка ICE candidates
        this.peerConnection.onicecandidate = (event) => {
            if (event.candidate) {
                this.signaling.send({
                    type: 'ice-candidate',
                    candidate: event.candidate
                });
            }
        };
        
        // Мониторинг состояния соединения
        this.peerConnection.onconnectionstatechange = () => {
            console.log('Connection state:', this.peerConnection.connectionState);
        };
        
        // Создание offer
        const offer = await this.peerConnection.createOffer();
        await this.peerConnection.setLocalDescription(offer);
        
        this.signaling.send({
            type: 'offer',
            sdp: offer.sdp
        });
    }
    
    async handleOffer(data) {
        await this.peerConnection.setRemoteDescription(
            new RTCSessionDescription({ type: 'offer', sdp: data.sdp })
        );
        
        const answer = await this.peerConnection.createAnswer();
        await this.peerConnection.setLocalDescription(answer);
        
        this.signaling.send({
            type: 'answer',
            sdp: answer.sdp
        });
    }
    
    async handleAnswer(data) {
        await this.peerConnection.setRemoteDescription(
            new RTCSessionDescription({ type: 'answer', sdp: data.sdp })
        );
    }
    
    async handleIceCandidate(data) {
        await this.peerConnection.addIceCandidate(
            new RTCIceCandidate(data.candidate)
        );
    }
    
    toggleAudio() {
        const audioTrack = this.localStream.getAudioTracks()[^0];
        if (audioTrack) {
            audioTrack.enabled = !audioTrack.enabled;
            return audioTrack.enabled;
        }
        return false;
    }
    
    toggleVideo() {
        const videoTrack = this.localStream.getVideoTracks()[^0];
        if (videoTrack) {
            videoTrack.enabled = !videoTrack.enabled;
            return videoTrack.enabled;
        }
        return false;
    }
    
    hangup() {
        if (this.localStream) {
            this.localStream.getTracks().forEach(track => track.stop());
        }
        if (this.peerConnection) {
            this.peerConnection.close();
        }
        this.signaling.close();
    }
}
```


#### 3.5.3 signaling.js — WebSocket клиент

```javascript
export class SignalingClient {
    constructor(roomId) {
        this.roomId = roomId;
        this.ws = null;
        this.handlers = new Map();
    }
    
    on(eventType, handler) {
        this.handlers.set(eventType, handler);
    }
    
    async connect() {
        return new Promise((resolve, reject) => {
            // Определяем протокол (ws или wss) на основе текущего протокола страницы
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws/${this.roomId}`;
            
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.send({ 
                    type: 'join', 
                    room_id: this.roomId 
                });
                resolve();
            };
            
            this.ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.handleMessage(message);
                } catch (error) {
                    console.error('Failed to parse message:', error);
                }
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                reject(error);
            };
            
            this.ws.onclose = () => {
                console.log('WebSocket closed');
            };
        });
    }
    
    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        } else {
            console.error('WebSocket is not connected');
        }
    }
    
    handleMessage(message) {
        console.log('Received message:', message.type);
        const handler = this.handlers.get(message.type);
        if (handler) {
            handler(message);
        }
    }
    
    close() {
        if (this.ws) {
            this.ws.close();
        }
    }
}
```


### 3.6 Нефункциональные требования

- Поддержка браузеров: Chrome 90+, Safari 14+, Firefox 88+
- Адаптивность для экранов от 320px до 4K
- Время загрузки страницы < 1 секунды
- Graceful degradation при отсутствии камеры/микрофона
- Обработка потери соединения с автопереподключением

***

## 4. Общие требования к MVP

### 4.1 Репозитории

Три отдельных Git-репозитория:[^18][^19]

- `kaamos-bot` — Telegram бот
- `kaamos-server` — WebRTC SFU сервер + статика фронтенда
- `kaamos-frontend` — Исходники фронтенда

**Обоснование:** Для MVP с тремя разными компонентами multirepo проще — независимая разработка, простой CI/CD, каждая команда может работать автономно.[^19][^18]

### 4.2 Деплой

- **Бот:** VPS с systemd service
- **Сервер:** VPS с Nginx reverse proxy (SSL через Let's Encrypt)
- **Фронтенд:** Статические файлы собираются в `dist/` и копируются в `kaamos-server/static/`


### 4.3 Мониторинг

- Логи в stdout (для сбора Docker/systemd)
- Health check endpoints: `/health` на сервере и боте
- Метрики: количество активных комнат, пользователей онлайн


### 4.4 Безопасность

- HTTPS обязателен для WebRTC[^15]
- Rate limiting на API endpoints (10 req/min на пользователя)
- Валидация room_id (regex: `^[a-zA-Z0-9]{10}$`)
- CORS настройки для фронтенда


### 4.5 Документация

- README в каждом репозитории с инструкциями по запуску
- API документация для сервера в формате OpenAPI 3.0
- Примеры запросов для тестирования с curl/Postman
<span style="display:none">[^20][^21][^22][^23][^24][^25][^26][^27][^28][^29][^30][^31][^32][^33][^34]</span>

<div align="center">⁂</div>

[^20]: https://www.youtube.com/watch?v=dxPakeBsgl4

[^21]: https://www.youtube.com/watch?v=d6s-cMlqLZc

[^22]: https://www.youtube.com/watch?v=tUkadNzfrl8

[^23]: https://www.youtube.com/watch?v=kwehxBDX_o8

[^24]: https://www.youtube.com/watch?v=anYyDvrmcUs

[^25]: https://www.youtube.com/watch?v=kcPnibD9yxI

[^26]: https://www.youtube.com/watch?v=rcmdyQL2DUM

[^27]: https://www.youtube.com/watch?v=UOnSWOC3LoE

[^28]: https://www.youtube.com/watch?v=24z0GjXC0Kg

[^29]: https://www.youtube.com/watch?v=DPsQg215Efo

[^30]: https://github.com/golang-standards/project-layout

[^31]: https://www.reddit.com/r/golang/comments/17w500a/best_practices_regarding_project_layout_package/

[^32]: https://webreference.com/go/best-practices/project-structure

[^33]: https://ldej.nl/post/structuring-go/

[^34]: https://www.aviator.co/blog/monorepo-a-hands-on-guide-for-managing-repositories-and-microservices/

