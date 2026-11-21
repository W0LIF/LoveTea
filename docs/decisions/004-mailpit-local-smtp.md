# ADR-004: Отладка почты через Mailpit

## Статус
Принято

## Контекст
В план 개발ы указан Mailpit для просмотра тестовых писем на порту 8025, но в исходном docker-compose.yml его не было.

## Решение
Добавлен сервис Mailpit в `infra/docker-compose.yml`. API настроен на отправку через Mailpit по SMTP (host: mailpit, port: 1025). В production заменяется на реальный SMTP сервер.
