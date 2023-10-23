kubectl apply -k operator
kubectl create ns monitoring
kubectl apply -k collector -n monitoring
kubectl apply -k instrumentation -n monitoring