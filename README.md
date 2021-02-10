# Otus Architect

Homework Otus Architect

## API
* в каталоге **examples** есть Postman коллекция
* в коллекции есть 2 каталога: Local (для локальной разработки) и K8s (для kubernetes - для корректной работы требуется явно передавать хост arch.homework в curl или прописать его в hosts)

## Развертывание через helm
* `helm install otus-architect-release ./deployments-helm/otus-architect`
    * релиз включает развеотывание сервиса 

## Устарело
### Манифесты для k8s
* манифесты находятся в каталоге **deployments**
* запуск командой **kubectl apply -f deployments**

