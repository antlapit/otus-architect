helm install nginx ingress-nginx/ingress-nginx -f deployments/nginx-ingress.yaml

helm install kafka bitnami/kafka -f deployments/kafka.yaml

helm install user-profile-service-release deployments-helm/user-profile-service
helm install auth-service-release deployments-helm/auth-service
helm install order-service-release deployments-helm/order-service
helm install billing-service-release deployments-helm/billing-service
helm install krakend deployments-helm/krakend

helm install prom prometheus-community/kube-prometheus-stack -f deployments/prometheus.yaml --atomic
helm install postgres-exporter-users prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-users.yaml
helm install postgres-exporter-auth prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-auth.yaml
