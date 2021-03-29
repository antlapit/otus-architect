helm delete user-profile-service-release
helm delete auth-service-release
helm delete krakend

helm delete prom
helm delete nginx
helm delete postgres-exporter-users
helm delete postgres-exporter-auth

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
