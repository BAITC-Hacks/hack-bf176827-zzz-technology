#!/bin/sh
# Проверяем среду до npm ci, чтобы Windows npm не запускался на пути WSL.
set -eu
if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
    echo 'Нужны Node.js 22.12+ и npm внутри этой Linux/WSL-системы. Windows Node.js не подходит для проекта в /home/.' >&2
    echo 'См. frontend/README.md, раздел «Node.js в WSL».' >&2
    exit 1
fi
if [ "$(uname -s)" = Linux ]; then
    case "$(command -v npm)" in
        /mnt/*|*.exe|*.cmd)
            echo 'Обнаружен Windows npm в Linux/WSL. Установите Linux Node.js и поставьте его bin первым в PATH.' >&2
            echo 'См. frontend/README.md, раздел «Node.js в WSL».' >&2
            exit 1
            ;;
    esac
    if [ "$(node -p 'process.platform')" != linux ]; then
        echo 'В Linux/WSL требуется Linux-версия Node.js.' >&2
        exit 1
    fi
fi
