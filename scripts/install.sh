helm install nginx ingress-nginx/ingress-nginx -f deployments/nginx-ingress.yaml

helm install postgres bitnami/postgresql -f deployments/postgresql.yaml
helm install kafka bitnami/kafka -f deployments/kafka.yaml
helm install mongodb bitnami/mongodb -f deployments/mongodb.yaml

helm install user-profile-service-release deployments-helm/user-profile-service
helm install auth-service-release deployments-helm/auth-service
helm install order-service-release deployments-helm/order-service
helm install billing-service-release deployments-helm/billing-service
helm install notification-service-release deployments-helm/notification-service
helm install product-service-release deployments-helm/product-service
helm install price-service-release deployments-helm/price-service
helm install product-search-service-release deployments-helm/product-search-service
helm install warehouse-service-release deployments-helm/warehouse-service
helm install delivery-service-release deployments-helm/delivery-service

helm install krakend deployments-helm/krakend

helm install prom prometheus-community/kube-prometheus-stack -f deployments/prometheus.yaml --atomic
helm install postgres-exporter-users prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-users.yaml
helm install postgres-exporter-auth prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-auth.yaml
helm install postgres-exporter-order prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-order.yaml
helm install postgres-exporter-billing prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-billing.yaml
helm install postgres-exporter-notification prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-notification.yaml
helm install postgres-exporter-product-search prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-product-search.yaml
helm install postgres-exporter-warehouse prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-warehouse.yaml
helm install postgres-exporter-delivery prometheus-community/prometheus-postgres-exporter -f deployments/postgresql-exporter-delivery.yaml
