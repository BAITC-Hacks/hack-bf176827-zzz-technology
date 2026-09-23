# React-интерфейс «Граф денег»

React + Vite + Cytoscape. Все зависимости входят в локальную сборку, внешних CDN нет.

## Docker

Из корня репозитория:

```sh
docker compose up --build
```

Откройте http://localhost:3000. Nginx раздаёт React и проксирует `/v1/` и
`/swagger/` в сервис `app:8080`. Резервный `/graph.json` читается из общего
каталога `out/`, поэтому доступен и при временной недоступности API.

## Разработка

Node.js 22.12+ (рекомендуется 24), npm; Go API должен быть запущен отдельно.

```sh
# Из корня, первый терминал:
make web
# Во втором терминале:
cd frontend
npm ci
npm run dev
```

Откройте http://localhost:5173. Для другого адреса API задайте
`API_PROXY_TARGET=http://127.0.0.1:8081 npm run dev`.

`npm run build` создаёт `dist/`, `npm test` проверяет операции с графом.
`npm run preview` раздаёт production-сборку с тем же API-прокси.
`?demo=1` явно включает тестовый граф из 10 узлов; реальная ошибка API
не подменяется моковыми данными.

`make demo` в корне собирает React, выполняет pipeline/check и запускает Go,
который раздаёт `frontend/dist/` на http://localhost:8080.

Компоненты: `App` — состояние и API, `GraphCanvas` — граф,
`NodeCard` — карточка, `Assistant` — опциональный LLM-интерфейс.
`graph.js` содержит операции с графом для резервного источника данных.

## Node.js в WSL

Для проекта в `/home/...` запускайте **Linux Node.js и npm внутри WSL**.
Windows npm из `/mnt/c/Program Files/nodejs/` вызывает CMD на UNC-пути
`\\wsl.localhost\Ubuntu\...`, что приводит к ошибке `C:\Windows\install.js`.

Проверка в терминале Ubuntu:

```sh
command -v node
command -v npm
node -p 'process.platform'  # должно быть linux
```

Если Node установлен в `~/.local/bin`, включите его в текущем терминале:

```sh
export PATH="$HOME/.local/bin:$PATH"
hash -r
cd ~/hackator
make frontend-build
```

`npm ci` самостоятельно пересоздаёт `node_modules` для текущей платформы;
не переносите этот каталог между Windows и Linux. `make frontend-build`
проверяет среду до установки и сообщает об ошибочном Windows npm.
