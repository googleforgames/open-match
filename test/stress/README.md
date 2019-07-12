This folder defines Open Match's stress test behaviors for release v0.6 and is not compatible to previous releases due to API conflicts. Open Match use [Locust](https://docs.locust.io/en/stable/) for load testing, please see the documentation if you wish to contribute more test senarios.

To run the stress test
```
# deploy Tiller to your GKE cluster
make push-helm
# deploy stress test services along with Open Match core components to the cluster, it may take a while because the cloud load balancer is trying to assign an external ip to the services in your cluster.
make install-ci-chart
# kubectl port-forward localhost traffic to the cluster
make proxy
# Run the front end stress test
make stress-frontend-[NUMBER_OF_PSEUDO_USERS]
# View the stress test result
cat stress_user[NUMBER_OF_PSEUDO_USERS]_requests.csv
cat stress_user[NUMBER_OF_PSEUDO_USERS]_distribution.csv
```