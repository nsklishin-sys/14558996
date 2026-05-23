# GitHub Actions — деплой LASTOP (автодеплой проверен 23.05)

## Когда срабатывает
- Любой push в ветку main
- Ручной запуск: Actions → Deploy to Yandex Cloud VM → Run workflow

## Что делает workflow
1. Проверяет что `go build ./...` проходит (защита от пушей с ошибками)
2. Подключается по SSH к VM в Yandex Cloud
3. На VM: git pull → go build → systemctl restart lastop
4. Health check /api/health — при провале автоматический откат на предыдущий коммит

## Требуемые GitHub Secrets

Заполнить в репо: Settings → Secrets and variables → Actions → New repository secret

- YC_VM_HOST — публичный IP VM (например 158.160.123.45). Берётся: Yandex Cloud Console → Compute → ВМ → детали
- YC_VM_USER — lastop. Это имя SSH-юзера, создан при настройке VM
- YC_VM_SSH_KEY — приватный ключ deploy-юзера. Целиком, начиная с -----BEGIN OPENSSH PRIVATE KEY----- и заканчивая -----END OPENSSH PRIVATE KEY-----

## Как сгенерировать ключ для GitHub Actions

ВАЖНО: НЕ использовать личный SSH-ключ разработчика. Для GitHub Actions нужен ОТДЕЛЬНЫЙ ключ, чтобы можно было его отозвать независимо.

На VM выполнить:

    ssh-keygen -t ed25519 -f ~/.ssh/github_actions_deploy -N "" -C "github-actions-deploy"
    cat ~/.ssh/github_actions_deploy.pub >> ~/.ssh/authorized_keys
    chmod 600 ~/.ssh/authorized_keys
    cat ~/.ssh/github_actions_deploy

Последняя команда покажет приватный ключ — его целиком копируем в GitHub Secret YC_VM_SSH_KEY. После добавления в Secret приватный ключ с VM можно удалить (он живёт только в GitHub).

## Sudo для systemctl restart

Юзер lastop должен иметь право на sudo systemctl restart lastop БЕЗ пароля.
На VM один раз выполнить sudo visudo и добавить в конец строку:

    lastop ALL=(ALL) NOPASSWD: /bin/systemctl restart lastop

Это разрешает ТОЛЬКО рестарт нашего сервиса, ничего больше.

## Откат вручную

На VM:

    cd /opt/lastop
    git log --oneline -10
    git reset --hard <commit_hash>
    go build -o /opt/lastop/bin/lastop ./cmd/server
    sudo systemctl restart lastop

## Проверка деплоя
- В GitHub: Actions → последний run должен быть зелёным
- На VM: sudo systemctl status lastop — active (running)
- В браузере: https://lastop.ru/api/health — {"status":"ok"}
