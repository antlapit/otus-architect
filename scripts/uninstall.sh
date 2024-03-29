helm delete user-profile-service-release
helm delete auth-service-release
helm delete order-service-release
helm delete billing-service-release
helm delete notification-service-release
helm delete product-service-release
helm delete price-service-release
helm delete product-search-service-release
helm delete warehouse-service-release
helm delete delivery-service-release
helm delete krakend

helm delete prom
helm delete nginx
helm delete postgres-exporter-users
helm delete postgres-exporter-auth
helm delete postgres-exporter-order
helm delete postgres-exporter-billing
helm delete postgres-exporter-notification
helm delete postgres-exporter-product-search
helm delete postgres-exporter-warehouse
helm delete postgres-exporter-delivery

helm delete kafka
helm delete postgres
helm delete mongodb

kubectl delete clusterrole nginx-ingress-nginx
kubectl delete clusterrole prom-grafana-clusterrole
kubectl delete clusterrole prom-kube-prometheus-stack-operator
kubectl delete clusterrole prom-kube-prometheus-stack-operator-psp
kubectl delete clusterrole prom-kube-prometheus-stack-prometheus
kubectl delete clusterrole prom-kube-prometheus-stack-prometheus-psp
kubectl delete clusterrole prom-kube-state-metrics
kubectl delete clusterrole psp-prom-kube-state-metrics
kubectl delete clusterrole psp-prom-prometheus-node-exporter


kubectl delete clusterrolebinding nginx-ingress-nginx
kubectl delete clusterrolebinding prom-grafana-clusterrole
kubectl delete clusterrolebinding prom-kube-prometheus-stack-operator
kubectl delete clusterrolebinding prom-kube-prometheus-stack-operator-psp
kubectl delete clusterrolebinding prom-kube-prometheus-stack-prometheus
kubectl delete clusterrolebinding prom-kube-prometheus-stack-prometheus-psp
kubectl delete clusterrolebinding prom-kube-state-metrics
kubectl delete clusterrolebinding psp-prom-kube-state-metrics
kubectl delete clusterrolebinding psp-prom-prometheus-node-exporter


kubectl delete job auth-service-release
kubectl delete job user-profile-service-release
kubectl delete job order-service-release
kubectl delete job billing-service-release
kubectl delete job notification-service-release
kubectl delete job product-service-release
kubectl delete job product-search-service-release
kubectl delete job price-service-release
