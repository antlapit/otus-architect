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

### Метрики с nginx
* обновление стандартного nginx `helm install nginx stable/nginx-ingress -f deployments/nginx-ingress.yaml`
