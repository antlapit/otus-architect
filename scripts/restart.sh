kubectl get pods | grep service | while read line ; do echo "$line" | awk '{print $1}' | sort | while read line2; do kubectl delete pod "$line2" ; done; done
