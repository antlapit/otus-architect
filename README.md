# Otus Architect

Homework Otus Architect

## API
* в каталоге **examples** есть Postman коллекция
* в коллекции есть 2 каталога: Local (для локальной разработки) и K8s (для kubernetes - для корректной работы требуется явно передавать хост arch.homework в curl или прописать его в hosts)

## Развертывание через helm
**Команда** `helm install otus-architect-release ./deployments-helm/otus-architect`

**Состав релиза**  
* развертывание postgresql из официального чарта
* развертывание сервиса
* выполнение стартовых миграций за счет job, который удаляется после релиза
  * миграции выполняются самим сервисом при наличии переменной окружения SERVICE_MODE=INIT
  * в БД создается 1 запись о пользователе с id = 1
  * докер образ для job и сервиса один
  
## Prometheus & Grafana
* установка Prometheus `helm install prom prometheus-community/kube-prometheus-stack -f deployments/prometheus.yaml --atomic`
* форвардинг портов grafana `kubectl port-forward service/prom-grafana 9000:80`
* dashboard grafana http://localhost:9000
* логин/пароль - admin/prom-operator
* форвардинг портов prometheus `kubectl port-forward service/prom-prometheus-operator-prometheus 9090`

### Дашборд Grafana
Дашборд находится в `grafana/dashboard.json`

Содержимое дашборда
* метрики с сервиса
  * Latency по квантилям 0.5, 0.95, 0.99, 1.0 с разбивкой по методам API
  * 5xx ошибки с разбивкой по методам API
  * RPS с разбивкой по методам API
* метрики с nginx
  * Latency по квантилям 0.5, 0.95, 0.99, 1.0 (+ alert-ы в Telegram)
  * 5xx ошибки (+ alert-ы в Telegram)
  * RPS
* метрики Postgres  
* CPU и Memory по pod-ам 

### Стресс-тестирование
* одновременный запуск скриптов `scripts/load_get.sh` и `scripts/load_delete.sh`
* для проверки 5хх ошибок во время тестирования имитировал "падение" БД


### Метрики с nginx
* обновление стандартного nginx `helm install nginx ingress-nginx/ingress-nginx -f deployments/nginx-ingress.yaml`

### Метрики с Postgres
