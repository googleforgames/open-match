# Knative instructions
This describes the basic installation of knative (and istio) for serverless match functions.
It also provides experimental instructions for setting up you function host and endpoint.

## New cluster deploy instructions
Primarily taken from this [https://github.com/knative/docs/blob/master/install/Knative-custom-install.md] guide 
### Install Istio
Download and apply the istio manifests
```
curl -L https://git.io/getLatestIstio | sh -
cd istio-1.0.5/
kubectl apply -f install/kubernetes/helm/istio/templates/crds.yaml
kubectl apply -f install/kubernetes/istio-demo-auth.yaml
```

Wait for `kubectl get pods --namespace istio-system` to complete

Don't forget, you'll need to apply istio-injection label for any namespace you deploy to
`kubectl label namespace default istio-injection=enabled`

### Install Knative
You may need to give your gcp user cluster admin privs to get through this part
`kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user="{your user}"`

To install knative, promethus + grafana (metrics), elk (logging), and zipkin (tracing), run. See the linked custom installation guide for more options
```
k apply -f https://github.com/knative/serving/releases/download/v0.3.0/serving.yaml /
        -f https://github.com/knative/serving/releases/download/v0.3.0/monitoring.yaml
```

Wait for `kubectl get pods --namespace knative-serving` to complete

## Serving
To use the knative serverless flow for match functions...
1. Create a host-harness for running your function as a hosted service at `/api/function` (roadmap v0.4 and v0.6 for something more official)
2. Modify the provided knative serving manifest example at /deployments/k8s/knative_sample.yaml with your image name
3. Run `kubectl apply -f knative_sample.yaml` to start up an instance of the function on your knative installation
4. Using the configurable Http/1.1 REST pattern in open match (in review calebatwd/knative-rest-mmf), specify a `{..."hostName": "{knative service name}", "port": 8080}` in your match object properties when calling CreateMatch. The default port is 8080 in knative, but you can use whatever you like.
5. This should tell open match mmforc to call your function via hostname discovery in kubernetes over the knative ingress. You should see logs from your mmf function.

### Other Notes
- After 5 minutes of inactivity, knative will spin down your function. Subsequent calls will have a warm-up period of a few seconds depending on your function and host.
- The default port for knative is 8080, but you can use whatever you like so long as the intended port is Docker EXPOSED, knative will bind on that
- Dns discovery over knative has been problematic via the ingress. If you experience issues, considering updating the dns host discovery in mmforc to resolve on the full name `function-name.default.svc.cluster.local`
- Http2 is currently not supported in knative, so GRPC is not currently usable.
- The REST contract for the function can be found in cmd/mmforc/main.go as 

```
	type Profile struct {
		JobName   string
		ProfId    string
		MoId      string
		PropId    string
		ResultsId string
		Timestamp string
	}
```